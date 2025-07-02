package main

import (
	"math"
	"strconv"
	"strings"
	"unicode"
)

func parse(re string, fromChoice bool) (astNode, int) {
	if len(re) == 0 {
		return nil, 0
	}

	if !fromChoice {
		choices, cons := parseChoices(re)
		if cons != 0 {
			return choices, cons
		}
	}

	group, cons := parseGroup(re)
	if cons != 0 {
		return group, cons
	}

	bracket, cons := parseBracket(re)
	if cons != 0 {
		return bracket, cons
	}

	return parseChar(re)
}

// ...|...|...
func parseChoices(re string) (*choices, int) {
	if len(re) == 0 {
		return nil, 0
	}

	var cs [][]astNode

	var children []astNode
	consumed := 0
	for len(re) > 0 {
		c, cons := parse(re, true)
		if cons == 0 {
			break
		}
		re = re[cons:]
		children = append(children, c)
		consumed += cons

		if len(re) > 0 && re[0] == '|' {
			consumed += 1
			re = re[1:]
			cs = append(cs, children)
			children = nil
		}
	}

	if children != nil {
		cs = append(cs, children)
	}

	// if we parsed just one, we are not a choices
	if len(cs) == 1 {
		return nil, 0
	}
	return &choices{Choices: cs}, consumed
}

// (...)
func parseGroup(re string) (*group, int) {
	if len(re) == 0 || re[0] != '(' {
		return nil, 0
	}

	// pop off '('
	re = re[1:]
	consumed := 1

	var children []astNode
	for {
		child, cons := parse(re, false)
		if cons == 0 {
			break
		}
		consumed += cons
		children = append(children, child)
		re = re[cons:]
	}

	if len(re) == 0 || re[0] != ')' {
		return nil, 0
	}

	// pop off ')'
	consumed += 1

	// see if there is a Quantifier
	re = re[1:]
	q, cons := parseQuantifier(re)
	return &group{
		Children:   children,
		Quantifier: q,
	}, consumed + cons
}

// [...] and [^...]
func parseBracket(re string) (*bracket, int) {
	if len(re) < 2 || re[0] != '[' {
		return nil, 0
	}

	// pop off '['
	consumed := 1
	re = re[1:]

	negate := re[0] == '^'
	if negate {
		consumed += 1
		re = re[1:]
	}

	var chars []byte
	var ranges []charRange

	var queue []byte
	for len(re) > 0 {
		r := re[0]
		queue = append(queue, r)
		consumed += 1
		re = re[1:]

		// reduce
		if len(queue) == 3 {
			if queue[1] == '-' &&
				(unicode.IsLetter(rune(queue[0])) || unicode.IsDigit(rune(queue[0]))) &&
				(unicode.IsLetter(rune(queue[2])) || unicode.IsDigit(rune(queue[2]))) {
				charRange := charRange{
					From: queue[0],
					To:   r,
				}
				ranges = append(ranges, charRange)
				queue = nil
			} else {
				chars = append(chars, queue[0])
				queue = queue[1:]
			}
		}

		// allow []...] or [^]...]
		if r == ']' && !(consumed == 2 || consumed == 3 && negate) {
			break
		}
	}

	// reduce the rest of the queue to just chars
	chars = append(chars, queue...)

	// need to distinct []...] and [^]...] from the empty bracket []...
	if chars[len(chars)-1] != ']' {
		if chars[0] == ']' {
			chars = nil
			consumed = 2
			if negate {
				consumed += 1
			}
		} else {
			return nil, 0
		}
	} else {
		chars = chars[:len(chars)-1]
	}

	if len(chars) == 0 {
		chars = nil
	}

	// see if there is a Quantifier
	q, cons := parseQuantifier(re)
	return &bracket{
		Negate:     negate,
		Ranges:     ranges,
		Chars:      chars,
		Quantifier: q,
	}, consumed + cons
}

// {m,n} and ? and * and +
func parseQuantifier(re string) (*quantifier, int) {
	if len(re) == 0 {
		return nil, 0
	}

	switch re[0] {
	case '+':
		return &quantifier{
			Min: 1,
			Max: math.MaxInt,
		}, 1
	case '?':
		return &quantifier{
			Min: 0,
			Max: 1,
		}, 1
	case '*':
		return &quantifier{
			Min: 0,
			Max: math.MaxInt,
		}, 1
	}

	if re[0] != '{' {
		return nil, 0
	}

	endIdx := strings.Index(re, "}")
	if endIdx == -1 {
		return nil, 0
	}
	// inside '{...}'
	re = re[1:endIdx]

	numStrs := strings.SplitN(re, ",", 2)

	occMin, err := strconv.Atoi(numStrs[0])
	if err != nil {
		return nil, 0
	}

	if len(numStrs) == 1 {
		return &quantifier{Min: occMin, Max: occMin}, 1 + endIdx
	}

	occMax, err := strconv.Atoi(numStrs[1])
	if err != nil {
		return nil, 0
	}

	// consume lastNum and }
	return &quantifier{
		Min: occMin,
		Max: occMax,
	}, 1 + endIdx
}

func parseChar(re string) (char, int) {
	if len(re) == 0 {
		return char{}, 0
	}

	if re[0] == '\\' {
		if len(re) >= 2 {
			c := re[1]
			re = re[2:]
			q, cons := parseQuantifier(re)
			return char{Char: c, Quantifier: q}, 2 + cons
		}
		return char{}, 0
	}

	// don't consume non-escaped meta characters
	switch re[0] {
	case '^', '$':
		// these don't work with quantifiers
		return char{Char: re[0]}, 1
	case '(', ')', '{', '}', '[', ']', '|', '?', '+', '*':
		return char{}, 0
	}

	c := re[0]
	re = re[1:]
	q, cons := parseQuantifier(re)
	return char{Char: c, Quantifier: q}, 1 + cons
}
