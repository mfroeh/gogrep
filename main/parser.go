package main

import (
	"math"
	"strconv"
	"strings"
)

func pointTo[T any](val T) *T {
	return &val
}

func parse(re string) (astNode, int) {
	if len(re) == 0 {
		return nil, 0
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
func parseChoice(re string) (*choice, int) {
	if len(re) == 0 {
		return nil, 0
	}

	var choices [][]astNode

	var children []astNode
	consumed := 0
	for {
		c, cons := parse(re)
		if cons == 0 {
			break
		}
		re = re[cons:]
		children = append(children, c)
		consumed += cons

		if re[0] == '|' {
			consumed += 1
			re = re[1:]
			choices = append(choices, children)
			children = nil
		}
	}

	if children != nil {
		choices = append(choices, children)
	}

	// if we parsed just one, we are not a choice
	if len(choices) == 0 {
		return nil, 0
	}
	return &choice{choices: choices}, consumed
}

// (...) or just any grouping without capturing (no parentheses)
func parseGroup(re string) (*group, int) {
	if len(re) == 0 || re[0] != '(' {
		return nil, 0
	}

	// pop off '('
	re = re[1:]
	consumed := 1

	var children []astNode
	for {
		child, cons := parse(re)
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
			if queue[1] == '-' {
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

	// pop off '{'
	consumed := 1
	re = re[1:]

	numStrs := strings.SplitN(re, ",", 2)
	if len(numStrs) != 2 {
		return nil, 0
	}

	occMin, err := strconv.Atoi(numStrs[0])
	if err != nil {
		return nil, 0
	}

	// consume first num and ','
	consumed += len(numStrs[0]) + 1

	endIdx := strings.Index(numStrs[1], "}")
	if endIdx == -1 {
		return nil, 0
	}

	occMax, err := strconv.Atoi(numStrs[1][:endIdx])
	if err != nil {
		return nil, 0
	}

	// consume lastNum and }
	consumed += endIdx + 1
	return &quantifier{
		Min: occMin,
		Max: occMax,
	}, consumed
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
	switch rune(re[0]) {
	case '(', ')', '{', '}', '[', ']', '|':
		return char{}, 0
	}

	c := re[0]
	re = re[1:]
	q, cons := parseQuantifier(re)
	return char{Char: c, Quantifier: q}, 1 + cons
}
