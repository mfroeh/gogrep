package main

import "fmt"

func FindSubmatch(re string, s string) ([]string, error) {
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

	reRoot, err := parseGroup("("+re+")", 0)
	if err != nil {
		return nil, fmt.Errorf("failed to construct regex from %q: %w", "("+re+")", err)
	}

	submatches := make([]string, 0)
	for i := range s {
		j, match := reRoot.match(s[i:], 0, 0)
		if match && (!strictEnd || j == len(s)) {
			return reRoot.collectSubmatches(s[i:], submatches), nil
		}

		if strictStart == true {
			break
		}
	}
	return nil, nil
}

func Match(re string, s string) (bool, error) {
	submatches, err := FindSubmatch(re, s)
	return len(submatches) > 0, err
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
