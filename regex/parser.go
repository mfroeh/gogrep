package regex

import (
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
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
		return nil, newParserError(j, "unexpected EOS", nil)
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

// [...] and [^...]
// this doesn't conform to POSIX, as we allow perl character sets, which mandates that '\' is not treated literally
// inside of bracket expressions, make sure to escape '^', '-', ']' and '\'
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

	// todo: those characters that are metacharacters inside of [...], such as '-' or ']', are interpreted literally at the front or back of the expression
	// we require them to be escape for now (not as pretty but semantically equivalent, just requires that you escape them)
	// this means that we will assume that an unescaped ']' means that the bracket expression is over

	var q []byte
	ranges := make([]charRange, 0)
	for j < len(re) && re[j] != ']' {
		if re[j] == '[' {
			rs, cons := parsePosixCharSet(re, j)
			if rs == nil {
				return nil, newParserError(j, "invalid POSIX character set", nil)
			}
			j += cons
			ranges = append(ranges, rs...)
		} else if re[j] == '\\' {
			rs := parsePerlCharSet(re, j)
			if rs != nil {
				ranges = append(ranges, rs...)
			} else {
				c := escapedChar(re[j+1])
				ranges = append(ranges, charRange{from: c, to: c})
			}
			j += 2
		} else {
			q = append(q, re[j])
			j += 1

			// reduce
			// todo: allow unescaped - at end and start
			if len(q) == 3 {
				if q[1] == '-' {
					ranges = append(ranges, charRange{from: q[0], to: q[2]})
					q = nil
				} else {
					ranges = append(ranges, charRange{from: q[0], to: q[0]})
					q = q[1:]
				}
			}
		}
	}

	if j >= len(re) || re[j] != ']' {
		return nil, newParserError(j, "unexpected EOS", nil)
	}

	for _, c := range q {
		ranges = append(ranges, charRange{from: c, to: c})
	}

	// pop off ]
	j += 1

	// see if there is a Quantifier
	mi, ma, cons, err := parseQuantifier(re, j)
	if err != nil {
		return nil, err
	}

	return &node{
		state: &bracketState{negate: negate, ranges: ranges},
		mi:    mi,
		ma:    ma,
		str:   re[i : j+cons],
	}, nil
}

func parsePosixCharSet(re string, i int) ([]charRange, int) {
	if i+8 < len(re) && re[i:i+8] == "[:word:]" {
		return []charRange{
			{from: 'a', to: 'z'},
			{from: 'A', to: 'Z'},
			{from: '0', to: '9'},
			{from: '_', to: '_'},
		}, 8
	}

	if i+9 < len(re) {
		s := re[i : i+9]
		switch s {
		case "[:alnum:]":
			return []charRange{
				{from: 'a', to: 'z'},
				{from: 'A', to: 'Z'},
				{from: '0', to: '9'},
			}, 9
		case "[:alpha:]":
			return []charRange{
				{from: 'a', to: 'z'},
				{from: 'A', to: 'Z'},
			}, 9
		case "[:ascii:]":
			return []charRange{
				{from: 0x0, to: 0x7f},
			}, 9
		case "[:blank:]":
			return []charRange{
				{from: ' ', to: ' '},
				{from: '\t', to: '\t'},
			}, 9
		case "[:cntrl:]":
			return []charRange{
				{from: 0x0, to: 0x1f},
				{from: 0x7f, to: 0x7f},
			}, 9
		case "[:digit:]":
			return []charRange{
				{from: '0', to: '9'},
			}, 9
		case "[:graph:]":
			return []charRange{
				{from: 0x21, to: 0x7e},
			}, 9
		case "[:lower:]":
			return []charRange{
				{from: 'a', to: 'z'},
			}, 9
		case "[:print:]":
			return []charRange{
				{from: 0x20, to: 0x7e},
			}, 9
		case "[:punct:]":
			return []charRange{
				{from: '[', to: '['},
				{from: ']', to: ']'},
				{from: '!', to: '!'},
				{from: '"', to: '"'},
				{from: '#', to: '#'},
				{from: '$', to: '$'},
				{from: '%', to: '%'},
				{from: '&', to: '&'},
				{from: '\'', to: '\''},
				{from: '(', to: '('},
				{from: ')', to: ')'},
				{from: '*', to: '*'},
				{from: '+', to: '+'},
				{from: ',', to: ','},
				{from: '.', to: '.'},
				{from: '/', to: '/'},
				{from: ':', to: ':'},
				{from: ';', to: ';'},
				{from: '<', to: '<'},
				{from: '=', to: '='},
				{from: '>', to: '>'},
				{from: '?', to: '?'},
				{from: '@', to: '@'},
				{from: '\\', to: '\\'},
				{from: '^', to: '^'},
				{from: '_', to: '_'},
				{from: '`', to: '`'},
				{from: '{', to: '{'},
				{from: '}', to: '}'},
				{from: '|', to: '|'},
				{from: '~', to: '~'},
				{from: '-', to: '-'},
			}, 9
		case "[:space:]":
			return []charRange{
				{from: ' ', to: ' '},
				{from: '\t', to: '\t'},
				{from: '\r', to: '\r'},
				{from: '\n', to: '\n'},
				{from: '\v', to: '\v'},
				{from: '\f', to: '\f'},
			}, 9
		case "[:upper:]":
			return []charRange{
				{from: 'A', to: 'Z'},
			}, 9
		}
	}

	if i+10 < len(re) && re[i:i+10] == "[:xdigit:]" {
		return []charRange{
			{from: 'A', to: 'F'},
			{from: 'a', to: 'f'},
			{from: '0', to: '9'},
		}, 10
	}

	return nil, 0
}

// supported: \w, \W, \d, \D, \s, \S
func parsePerlCharSet(re string, i int) []charRange {
	if i+1 < len(re) {
		s := re[i : i+2]
		switch s {
		case `\w`, `\W`:
			ranges := []charRange{
				{from: 'a', to: 'z'},
				{from: 'A', to: 'Z'},
				{from: '0', to: '9'},
				{from: '_', to: '_'},
			}
			if s == `\W` {
				return negateCharRanges(ranges)
			}
			return ranges
		case `\d`, `\D`:
			ranges := []charRange{
				{from: '0', to: '9'},
			}
			if s == `\D` {
				return negateCharRanges(ranges)
			}
			return ranges
		case `\s`, `\S`:
			ranges := []charRange{
				{from: ' ', to: ' '},
				{from: '\t', to: '\t'},
				{from: '\r', to: '\r'},
				{from: '\n', to: '\n'},
				{from: '\v', to: '\v'},
				{from: '\f', to: '\f'},
			}
			if s == `\S` {
				return negateCharRanges(ranges)
			}
			return ranges
		}
	}

	return nil
}

func parseChar(re string, i int) (*node, error) {
	if i >= len(re) {
		return nil, nil
	}

	if re[i] != '\\' {
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

		c := re[i]
		node := &node{
			state: &charState{char: c},
			mi:    mi,
			ma:    ma,
			str:   re[i : i+1+cons],
		}
		// have to differentiate between literal '\.' and wildcard '.'
		if re[i] == '.' {
			node.state = &bracketState{
				negate: true,
				ranges: []charRange{{from: '\n', to: '\n'}},
			}
		}
		return node, nil
	}

	// if re[i] == '\'
	if i+1 < len(re) {
		// we always want to parse a quantifier
		mi, ma, cons, err := parseQuantifier(re, i+2)
		if err != nil {
			return nil, err
		}

		// try to parse perl char set
		charSet := parsePerlCharSet(re, i)
		if charSet != nil {
			return &node{state: &bracketState{ranges: charSet}, mi: mi, ma: ma, str: re[i : i+2+cons]}, nil
		}

		// otherwise treat as an escaped literal
		return &node{
			state: &charState{char: escapedChar(re[i+1])},
			mi:    mi,
			ma:    ma,
			str:   re[i : i+2+cons],
		}, nil
	}
	return nil, newParserError(i, "unexpected EOS", nil)
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

func negateCharRanges(ranges []charRange) []charRange {
	newRanges := make([]charRange, len(ranges))

	// assumes that intervals don't overlap
	slices.SortFunc(ranges, func(a, b charRange) int {
		return int(a.from) - int(b.from)
	})

	from := byte(0)
	for _, r := range ranges {
		newRanges = append(newRanges, charRange{from: from, to: r.from - 1})
		from = r.to + 1
	}
	newRanges = append(newRanges, charRange{from: from, to: 0x7f})
	return newRanges
}

// parse an ASCII escape sequence from c if there is one (e.g. '\t', '\n', ...)
// if c isn't an ASCII escape sequence, return c
// should be called if the character preceding c in the input string is '\'
// https://en.wikipedia.org/wiki/Escape_sequences_in_C
func escapedChar(c byte) byte {
	switch c {
	case 'a':
		return '\a'
	case 'b':
		return '\b'
	case 'e':
		// funnily enough '\e' does not exist in golang :D
		return 0xb
	case 'f':
		return '\f'
	case 'n':
		return '\n'
	case 'r':
		return '\r'
	case 't':
		return '\t'
	case 'v':
		return '\v'
	}
	return c
}
