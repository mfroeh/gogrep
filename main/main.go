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

	reRoot, ok := parseGroup("(" + re + ")")
	if ok == 0 {
		return false, fmt.Errorf("failed to construct regex from %q", re)
	}

	if strictStart {
		offsets := reRoot.match(s, []int{0})
		if len(offsets) > 0 {
			if strictEnd {
				return offsets[0] == len(s), nil
			}
			return true, nil
		}
		return false, nil
	}

	for i := range s {
		offsets := reRoot.match(s[i:], []int{0})
		if len(offsets) > 0 {
			if strictEnd {
				return offsets[0] == len(s[i:]), nil
			}
			return true, nil
		}
	}
	return false, nil
}

func main() {
	println("Hello World")
}
