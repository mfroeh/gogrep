package main

import "fmt"

func Match(re string, s string) (bool, error) {
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
	_ = strictEnd

	reRoot, err := parseGroup("("+re+")", 0)
	if err != nil {
		return false, fmt.Errorf("failed to construct regex from %q: %w", "("+re+")", err)
	}

	if strictStart {
		_, match := reRoot.match(s, 0, 0)
		if match {
			return true, nil
		}
		return false, nil
	}

	for i := range s {
		_, match := reRoot.match(s[i:], 0, 0)
		if match {
			return true, nil
		}
	}
	return false, nil
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
