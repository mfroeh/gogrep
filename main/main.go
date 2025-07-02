package main

import "fmt"

func Match(re string, s string) (bool, error) {
	regexp, ok := parseGroup("(" + re + ")")
	if ok == 0 {
		return false, fmt.Errorf("failed to construct regex from %q", re)
	}

	for i := range s {
		if len(regexp.match(s[i:], []int{0})) > 0 {
			return true, nil
		}
	}
	return false, nil
}

func main() {
	println("Hello World")
}
