package main

import (
	"log"
	"math"
	"slices"
)

var quantifierOptional = &quantifier{Min: 0, Max: 1}
var quantifierPlus = &quantifier{Min: 1, Max: math.MaxInt}
var quantifierMul = &quantifier{Min: 0, Max: math.MaxInt}

// {m,n} : {m, n}
// ? : {0, 1}
// * : {0, infty}
// + : {1, infty}
type quantifier struct {
	Min int
	Max int
}

type astNode interface {
	match(s string, offsets []int) []int
}

// ()
type group struct {
	Children   []astNode
	Quantifier *quantifier
}

func (g *group) match(s string, offsets []int) []int {
	if len(offsets) == 0 {
		log.Fatalln("got 0 offsets")
	}

	minCount := 1
	maxCount := 1
	if g.Quantifier != nil {
		minCount = g.Quantifier.Min
		maxCount = g.Quantifier.Max
	}

	var matches []int
	for _, off := range offsets {
		if minCount == 0 {
			matches = append(matches, off)
		}

		sOff := s[off:]
	q:
		for j := 0; j < maxCount; {
			childOffsets := []int{0}
			for _, c := range g.Children {
				childOffsets = c.match(sOff, childOffsets)
				if len(childOffsets) == 0 {
					break q
				}
			}
			matchOff := slices.Max(childOffsets)
			sOff = sOff[matchOff:]

			j++
			if j >= minCount {
				matches = append(matches, off+matchOff)
			}
		}
	}

	return matches
}

// |
type choices struct {
	Choices [][]astNode
}

func (c *choices) match(s string, offsets []int) []int {
	if len(offsets) == 0 {
		log.Fatalln("got 0 offsets")
	}

	var matches []int
	for _, off := range offsets {
		sOff := s[off:]
	cs:
		for _, choice := range c.Choices {
			childOffsets := []int{0}
			for _, c := range choice {
				childOffsets = c.match(sOff, childOffsets)
				if len(childOffsets) == 0 {
					continue cs
				}
			}
			matchOff := slices.Max(childOffsets)
			sOff = sOff[matchOff:]
			matches = append(matches, off+matchOff)
		}
	}

	slices.Reverse(matches)
	return matches
}

// []
type bracket struct {
	Negate     bool
	Chars      []byte
	Ranges     []charRange
	Quantifier *quantifier
}

func (b *bracket) match(s string, offsets []int) []int {
	if len(offsets) == 0 {
		log.Fatalln("got 0 offsets")
	}

	minCount := 1
	maxCount := 1
	if b.Quantifier != nil {
		minCount = b.Quantifier.Min
		maxCount = b.Quantifier.Max
	}

	var matches []int

	for _, off := range offsets {
		if minCount == 0 {
			matches = append(matches, off)
		}

		sOff := s[off:]
		for i := 0; i < min(maxCount, len(sOff)); {
			matchesChar := slices.Contains(b.Chars, sOff[i])
			if matchesChar && b.Negate {
				break
			}

			matchesRange := slices.ContainsFunc(b.Ranges, func(r charRange) bool { return r.inRange(sOff[i]) })
			if matchesRange && b.Negate {
				break
			}

			if !matchesChar && !matchesRange && !b.Negate {
				break
			}

			i++
			if i >= minCount {
				matches = append(matches, off+i)
			}
		}
	}

	slices.Reverse(matches)
	return matches
}

type charRange struct {
	// a-Z
	From byte
	To   byte
}

func (r charRange) inRange(c byte) bool {
	return c >= r.From && c <= r.To
}

type char struct {
	Char       byte
	Quantifier *quantifier
}

// todo: ^ and $ handling
func (c char) match(s string, offsets []int) []int {
	if len(offsets) == 0 {
		log.Fatalln("got 0 offsets")
	}

	minCount := 1
	maxCount := 1
	if c.Quantifier != nil {
		minCount = c.Quantifier.Min
		maxCount = c.Quantifier.Max
	}

	var matches []int
	for _, off := range offsets {
		if minCount == 0 {
			matches = append(matches, off+0)
		}

		sOff := s[off:]
		for i := 0; i < min(maxCount, len(sOff)); {
			if sOff[i] == c.Char || c.Char == '.' {
				i++
				if i >= minCount {
					matches = append(matches, off+i)
				}
			} else {
				break
			}
		}
	}

	slices.Reverse(matches)
	return matches
}
