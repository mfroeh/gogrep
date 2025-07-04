package main

import (
	"fmt"
	"strings"
	"unicode"
)

type Regex struct {
	root        *node
	strictStart bool
	strictEnd   bool
}

type Submatch struct {
	Offset int
	Str    string
}

func Compile(re string) (Regex, error) {
	strictStart := false
	if len(re) > 0 && re[0] == '^' {
		strictStart = true
		re = re[1:]
	}

	strictEnd := false
	if len(re) > 0 && re[len(re)-1] == '$' {
		strictEnd = true
		re = re[:len(re)-1]
	}

	root, err := parseGroup("("+re+")", 0)
	if err != nil {
		return Regex{}, fmt.Errorf("failed to construct regex from %q: %w", "("+re+")", err)
	}
	return Regex{
		root:        root,
		strictStart: strictStart,
		strictEnd:   strictEnd,
	}, nil
}

func (re Regex) FindSubmatch(s string) []Submatch {
	for i := range s {
		_, match := re.root.match(s[i:], 0, 0)
		if match {
			var submatches []Submatch
			re.root.collectSubmatches(s[i:], &submatches)
			for j := range submatches {
				submatches[j].Offset += i
			}
			if !re.strictEnd || len(submatches[0].Str) == len(s[i:]) {
				return submatches
			}
		}

		if re.strictStart == true {
			break
		}
	}
	return nil
}

func (re Regex) Match(s string) bool {
	return len(re.FindSubmatch(s)) > 0
}

func (re Regex) Replace(s string, with string) string {
	submatches := re.FindSubmatch(s)
	out := strings.Builder{}
	for i := 0; i < len(with); i++ {
		if with[i] == '$' && i+1 < len(with) && unicode.IsDigit(rune(with[i+1])) {
			num := 0
			for j := i + 1; j < len(with) && unicode.IsDigit(rune(with[j])); j++ {
				num *= 10
				num += int(with[j] - '0')
				i++
			}

			if num < len(submatches) {
				out.WriteString(submatches[num].Str)
			}
		} else {
			out.WriteByte(with[i])
		}
	}
	return out.String()
}

func main() {
	// missing and I want to add:
	// fill capture groups to enable find/replace
	// support POSIX char sets (e.g. \d and \D)
	// build a small GREP tool on top of this library
	// multiline support (in particular with $)
	// find all matches, replace all matches
	// potentially: unicode support
	// potentially: more than just ERE support, e.g. non-greedy (lazy) quantifier variants like .+?
	// potentially: look ahead/look behind
	println("Hello World")
}
