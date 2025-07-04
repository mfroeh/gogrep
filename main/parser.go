package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"
)

type parserError struct {
	inner   error
	message string
}

func (p parserError) Error() string {
	return p.message
}

func newParserError(i int, str string, inner error) parserError {
	return parserError{message: fmt.Sprintf("parser error at %d: %s", i, str), inner: inner}
}

func parse(re string, i int, fromChoice bool, prev *node) (*node, error) {
	if i >= len(re) {
		return nil, nil
	}

	if !fromChoice {
		choices, err := parseChoices(re, i)
		if err != nil {
			return nil, err
		}
		if choices != nil {
			if prev != nil {
				prev.next = choices
			}
			return choices, nil
		}
	}

	// group
	group, err := parseGroup(re, i)
	if err != nil {
		return nil, err
	}
	if group != nil {
		if prev != nil {
			prev.next = group
		}
		return group, nil
	}

	// bracket
	bracket, err := parseBracket(re, i)
	if err != nil {
		return nil, err
	}
	if bracket != nil {
		if prev != nil {
			prev.next = bracket
		}
		return bracket, nil
	}

	// char
	char, err := parseChar(re, i)
	if err != nil {
		return nil, err
	}
	if char != nil {
		if prev != nil {
			prev.next = char
		}
		return char, nil
	}
	return nil, nil
}

// ...|...|...
func parseChoices(re string, i int) (*node, error) {
	if i >= len(re) {
		return nil, nil
	}

	var choices []*node

	j := i
	var firstChild *node
	var prevChild *node
	for j < len(re) {
		child, err := parse(re, j, true, prevChild)
		if err != nil {
			return nil, err
		}
		if child == nil {
			break
		}
		j += len(child.str)
		prevChild = child
		if firstChild == nil {
			firstChild = child
		}

		if j <= len(re) && re[j] == '|' {
			j += 1
			choices = append(choices, firstChild)
			firstChild = nil
			prevChild = nil
		}
	}

	if firstChild != nil {
		choices = append(choices, firstChild)
	}

	// if we parsed just one, we are not a choice
	if len(choices) <= 1 {
		return nil, nil
	}
	return &node{
		state: &choiceState{
			choices: choices,
		},
		mi:  1,
		ma:  1,
		str: re[i:j],
	}, nil
}

// (...)
func parseGroup(re string, i int) (*node, error) {
	if i >= len(re) || re[i] != '(' {
		return nil, nil
	}

	// pop off '('
	j := i + 1

	var firstChild *node
	var prevChild *node
	for j < len(re) && re[j] != ')' {
		child, err := parse(re, j, false, prevChild)
		if err != nil {
			return nil, err
		}
		if child == nil {
			break
		}
		j += len(child.str)
		prevChild = child
		if firstChild == nil {
			firstChild = child
		}
	}

	if j >= len(re) {
		return nil, newParserError(j, "unexpected EOF", nil)
	}

	if re[j] != ')' {
		return nil, newParserError(j, "did not find closing ')'", nil)
	}

	// pop off ')'
	j++

	// see if there is a Quantifier
	mi, ma, cons, err := parseQuantifier(re, j)
	if err != nil {
		return nil, err
	}
	return &node{
		state: &groupState{
			firstChild: firstChild,
		},
		mi:  mi,
		ma:  ma,
		str: re[i : j+cons],
	}, nil
}

// todo: make this more sensible
// [...] and [^...]
func parseBracket(re string, i int) (*node, error) {
	if i >= len(re) {
		return nil, nil
	}

	if re[i] != '[' {
		return nil, nil
	}

	// pop off '['
	j := i + 1

	negate := re[j] == '^'
	if negate {
		j++
	}

	var chars []byte
	var ranges []charRange

	var queue []byte
	for j < len(re) {
		r := re[j]
		queue = append(queue, r)
		j++

		// reduce
		if len(queue) == 3 {
			if queue[1] == '-' &&
				(unicode.IsLetter(rune(queue[0])) || unicode.IsDigit(rune(queue[0]))) &&
				(unicode.IsLetter(rune(queue[2])) || unicode.IsDigit(rune(queue[2]))) {
				charRange := charRange{
					from: queue[0],
					to:   r,
				}
				ranges = append(ranges, charRange)
				queue = nil
			} else {
				chars = append(chars, queue[0])
				queue = queue[1:]
			}
		}

		// allow []...] or [^]...]
		if r == ']' && !(j == i+2 || j == i+3 && negate) {
			break
		}
	}

	// reduce the rest of the queue to just chars
	chars = append(chars, queue...)

	// need to distinct []...] and [^]...] from the empty bracket []...
	if chars[len(chars)-1] != ']' {
		if chars[0] == ']' {
			chars = nil
			j = i + 2
			if negate {
				j++
			}
		} else {
			return nil, newParserError(i, "did not find closing ']'", nil)
		}
	} else {
		chars = chars[:len(chars)-1]
	}

	for _, c := range chars {
		ranges = append(ranges, charRange{from: c, to: c})
	}

	// see if there is a Quantifier
	mi, ma, cons, err := parseQuantifier(re, j)
	if err != nil {
		return nil, err
	}
	return &node{
		state: &bracketState{
			negate: negate,
			ranges: ranges,
		},
		mi:  mi,
		ma:  ma,
		str: re[i : j+cons],
	}, nil
}

func parseChar(re string, i int) (*node, error) {
	if i >= len(re) {
		return nil, nil
	}

	if re[i] == '\\' {
		if i+1 < len(re) {
			mi, ma, cons, err := parseQuantifier(re, i+2)
			if err != nil {
				return nil, err
			}
			return &node{
				state: &charState{char: re[i+1]},
				mi:    mi,
				ma:    ma,
				str:   re[i : i+2+cons],
			}, nil
		}
		return nil, newParserError(i, "unexpected EOF", nil)
	}

	// don't consume non-escaped meta characters
	switch re[i] {
	case '^', '$':
		return nil, newParserError(i, "unexpected meta character", nil)
	case '(', ')', '{', '}', '[', ']', '|', '?', '+', '*':
		return nil, nil
	}

	mi, ma, cons, err := parseQuantifier(re, i+1)
	if err != nil {
		return nil, err
	}

	// have to differentiate between literal '\.' and wildcard '.'
	c := re[i]
	if re[i] == '.' {
		c = wildcardChar
	}
	return &node{
		state: &charState{char: c},
		mi:    mi,
		ma:    ma,
		str:   re[i : i+1+cons],
	}, nil
}

// {m, n} and ? and * and +
func parseQuantifier(re string, i int) (mi int, ma int, consumed int, err error) {
	if i >= len(re) {
		return 1, 1, 0, nil
	}

	switch re[i] {
	case '+':
		return 1, math.MaxInt, 1, nil
	case '?':
		return 0, 1, 1, nil
	case '*':
		return 0, math.MaxInt, 1, nil
	}

	if re[i] != '{' {
		return 1, 1, 0, nil
	}

	re = re[i:]
	endIdx := strings.Index(re, "}")
	if endIdx == -1 {
		return 0, 0, 0, newParserError(i, "did not find closing '}'", nil)
	}

	// inside '{...}'
	re = re[1:endIdx]

	numStrs := strings.SplitN(re, ",", 2)

	occMin, err := strconv.Atoi(numStrs[0])
	if err != nil {
		return 0, 0, 0, newParserError(i, "failed to convert to number", err)
	}

	if len(numStrs) == 1 {
		return occMin, occMin, 1 + endIdx, nil
	}

	occMax, err := strconv.Atoi(numStrs[1])
	if err != nil {
		return 0, 0, 0, newParserError(i, "failed to convert to number", err)
	}

	return occMin, occMax, 1 + endIdx, nil
}
