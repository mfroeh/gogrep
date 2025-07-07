package regex

import (
	"slices"
)

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
	match      *Submatch
}

type node struct {
	state any
	mi    int
	ma    int
	next  *node
	str   string
}

func (n *node) match(in string, i, r int) (int, bool) {
	if n != nil {
		//fmt.Fprintf(os.Stdout, "%s, %s, %d, %d\n", in, n.str, i, r)
	}

	if n == nil {
		return i, true
	}

	j := i
	matchFound := false
	switch s := n.state.(type) {
	case *charState:
		if i >= len(in) {
			if r < n.mi {
				return i, false
			}
			return n.next.match(in, i, 0)
		}

		if in[i] == s.char {
			j++
			matchFound = true
		}
	case *bracketState:
		if i >= len(in) {
			if r < n.mi {
				return i, false
			}
			return n.next.match(in, i, 0)
		}

		if slices.ContainsFunc(s.ranges, func(r charRange) bool { return r.inRange(in[i]) }) {
			if !s.negate {
				matchFound = true
				j++
			}
		} else if s.negate {
			matchFound = true
			j++
		}
	case *choiceState:
		for _, c := range s.choices {
			j, matchFound := c.match(in, i, 0)
			if matchFound {
				return j, matchFound
			}
		}
		return i, false
	case *groupState:
		j, matchFound = s.firstChild.match(in, i, 0)
	default:
		panic("unexpected `state` type")
	}

	if matchFound {
		// repeat current expression (greedy)
		if r+1 < n.ma {
			end, match := n.match(in, j, r+1)
			if match {
				if g, ok := n.state.(*groupState); ok && g.match == nil {
					g.match = &Submatch{
						Offset: i,
						Str:    in[i:j],
					}
				}
				return end, match
			}
		}

		// advance to next expression
		if r+1 >= n.mi {
			end, match := n.next.match(in, j, 0)
			if match {
				if g, ok := n.state.(*groupState); ok && g.match == nil {
					g.match = &Submatch{
						Offset: i,
						Str:    in[i:j],
					}
				}
				return end, match
			}
		}
	}

	// for ? and *, also jump to next expression once without consuming the current match
	if n.mi == 0 && r == 0 {
		end, match := n.next.match(in, i, 0)
		if g, ok := n.state.(*groupState); ok && g.match == nil {
			g.match = &Submatch{
				Offset: i,
				Str:    "",
			}
		}
		return end, match
	}

	return i, false
}

func (n *node) collectSubmatches(in string, submatches *[]Submatch) {
	if n == nil {
		return
	}

	switch s := n.state.(type) {
	case *groupState:
		if s.match != nil {
			*submatches = append(*submatches, *s.match)
			// reset state so that we can reuse regex
			s.match = nil
		}
		s.firstChild.collectSubmatches(in, submatches)
	case *choiceState:
		for _, c := range s.choices {
			c.collectSubmatches(in, submatches)
		}
	}
	n.next.collectSubmatches(in, submatches)
}

type charRange struct {
	from byte
	to   byte
}

func (r charRange) inRange(c byte) bool {
	return c >= r.from && c <= r.to
}
