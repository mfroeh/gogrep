package main

import "slices"

type charState struct {
	char byte
}

type bracketState struct {
	negate bool
	ranges []charRange
}

type choiceState struct {
	choices []*node
}

type groupState struct {
	firstChild *node
}

type node struct {
	state any
	mi    int
	ma    int
	next  *node
}

func (n *node) match(in string, i, r int) (int, bool) {
	if n == nil {
		return 0, true
	}

	if i >= len(in) {
		if r < n.mi {
			return 0, false
		}
		return n.next.match(in, i, 0)
	}

	var (
		adv        int
		matchFound bool
	)
	switch s := n.state.(type) {
	case *charState:
		if in[i] == s.char || s.char == '.' {
			adv = 1
			matchFound = true
		}
	case *bracketState:
		if slices.ContainsFunc(s.ranges, func(r charRange) bool { return r.inRange(in[i]) }) {
			if !s.negate {
				matchFound = true
				adv = 1
			}
		} else if s.negate {
			matchFound = true
			adv = 1
		}
	case *choiceState:
		for _, c := range s.choices {
			adv, matchFound = c.match(in, i, 0)
			if matchFound {
				return adv, matchFound
			}
		}
		return 0, false
	case *groupState:
		adv, matchFound = s.firstChild.match(in, i, 0)
	default:
		panic("unexpected `state` type")
	}

	if !matchFound {
		// special case for *, ? to avoid infinite recursion
		if n.mi == 0 {
			return n.next.match(in, i, 0)
		}
		return 0, false
	}

	// try to match again after advancing
	if r+1 < n.ma {
		repAdv, repMatch := n.match(in, i+adv, r+1)
		if repMatch {
			return repAdv, repMatch
		}
	}

	return n.next.match(in, i+adv, 0)
}
