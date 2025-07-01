package main

import (
	"github.com/google/go-cmp/cmp"
	"math"
	"testing"
)

func TestParseChoices(t *testing.T) {
	/*	tests := map[string]struct {
			givenStr     string
			wantChoices  choice
			wantConsumed int
		}{
			"happy two literal choices": {
				givenStr:    "abc|def",
				wantChoices: choice{choices: []astNode{}},
			},
		}

		_ = tests*/
}

func TestParseGroup(t *testing.T) {
	tests := map[string]struct {
		givenStr     string
		wantGroup    *group
		wantConsumed int
	}{
		"dont parse if no group opening": {
			givenStr: ")abc",
		},
		"dont parse if no group closing": {
			givenStr: "(abc",
		},
		"happy parse group of literal": {
			givenStr: "(ab)",
			wantGroup: &group{
				Children: []astNode{char{Char: 'a'}, char{Char: 'b'}},
			},
			wantConsumed: 4,
		},
		"happy parse group of literal and bracket": {
			givenStr: "(ab?[a-Z]?)+",
			wantGroup: &group{
				Children: []astNode{
					char{Char: 'a'}, char{Char: 'b', Quantifier: &quantifier{Min: 0, Max: 1}},
					&bracket{Negate: false, Ranges: []charRange{{From: 'a', To: 'Z'}}, Quantifier: &quantifier{Min: 0, Max: 1}},
				},
				Quantifier: &quantifier{Min: 1, Max: math.MaxInt},
			},
			wantConsumed: 12,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// when
			gotGroup, gotConsumed := parseGroup(tt.givenStr)

			// then
			if d := cmp.Diff(tt.wantGroup, gotGroup); d != "" {
				t.Errorf("diff (-want +got):\n%s", d)
			}

			if d := cmp.Diff(tt.wantConsumed, gotConsumed); d != "" {
				t.Errorf("consumed diff (-want +got):\n%s", d)
			}
		})
	}
}

func TestParseBracket(t *testing.T) {
	tests := map[string]struct {
		givenStr     string
		wantBracket  *bracket
		wantConsumed int
	}{
		"dont parse if no opening 1": {
			givenStr: "]abc",
		},
		"dont parse if no opening 2": {
			givenStr: "a[c]",
		},
		"happy empty": {
			givenStr: "[]",
			wantBracket: &bracket{
				Negate: false,
				Chars:  nil,
				Ranges: nil,
			},
			wantConsumed: 2,
		},
		"happy with Quantifier": {
			givenStr: "[a-Z]{1,3}",
			wantBracket: &bracket{
				Negate: false,
				Chars:  nil,
				Ranges: []charRange{{From: 'a', To: 'Z'}},
				Quantifier: &quantifier{
					Min: 1,
					Max: 3,
				},
			},
			wantConsumed: 10,
		},
		"happy don't escape": {
			givenStr: "[\\n]",
			wantBracket: &bracket{
				Negate: false,
				Chars:  []byte{'\\', 'n'},
				Ranges: nil,
			},
			wantConsumed: 4,
		},
		"happy empty with junk after": {
			givenStr: "[]abasdfsadl",
			wantBracket: &bracket{
				Negate: false,
				Chars:  nil,
				Ranges: nil,
			},
			wantConsumed: 2,
		},
		"happy empty negated with junk after": {
			givenStr: "[^]abasdfsadl",
			wantBracket: &bracket{
				Negate: true,
				Chars:  nil,
				Ranges: nil,
			},
			wantConsumed: 3,
		},
		"happy with junk after": {
			givenStr: "[a]abasdfsadl",
			wantBracket: &bracket{
				Negate: false,
				Chars:  []byte{'a'},
				Ranges: nil,
			},
			wantConsumed: 3,
		},
		"happy negated with junk after": {
			givenStr: "[^a]abasdfsadl",
			wantBracket: &bracket{
				Negate: true,
				Chars:  []byte{'a'},
				Ranges: nil,
			},
			wantConsumed: 4,
		},
		"happy negated ^": {
			givenStr: "[^]",
			wantBracket: &bracket{
				Negate: true,
				Chars:  nil,
				Ranges: nil,
			},
			wantConsumed: 3,
		},
		"happy Char range and chars": {
			givenStr: "[aa-ZbCd]",
			wantBracket: &bracket{
				Negate: false,
				Chars:  []byte{'a', 'b', 'C', 'd'},
				Ranges: []charRange{
					{
						From: 'a',
						To:   'Z',
					},
				},
			},
			wantConsumed: 9,
		},
		"happy Char range and '-' literal start": {
			givenStr: "[-a-Z]",
			wantBracket: &bracket{
				Negate: false,
				Chars:  []byte{'-'},
				Ranges: []charRange{
					{
						From: 'a',
						To:   'Z',
					},
				},
			},
			wantConsumed: 6,
		},
		"happy Char range and '-' literal end": {
			givenStr: "[a-Z-]",
			wantBracket: &bracket{
				Negate: false,
				Chars:  []byte{'-'},
				Ranges: []charRange{
					{
						From: 'a',
						To:   'Z',
					},
				},
			},
			wantConsumed: 6,
		},
		"happy negated Char range": {
			givenStr: "[^a-Z]b",
			wantBracket: &bracket{
				Negate: true,
				Ranges: []charRange{
					{
						From: 'a',
						To:   'Z',
					},
				},
			},
			wantConsumed: 6,
		},
		"happy negated Char range and '^' literal": {
			givenStr: "[^^a-Z]",
			wantBracket: &bracket{
				Negate: true,
				Chars:  []byte{'^'},
				Ranges: []charRange{
					{
						From: 'a',
						To:   'Z',
					},
				},
			},
			wantConsumed: 7,
		},
		"] can be included directly after [": {
			givenStr: "[]a-Z]",
			wantBracket: &bracket{
				Negate: false,
				Chars:  []byte{']'},
				Ranges: []charRange{
					{
						From: 'a',
						To:   'Z',
					},
				},
			},
			wantConsumed: 6,
		},
		"] can be included directly after [^": {
			givenStr: "[^]a-Z]",
			wantBracket: &bracket{
				Negate: true,
				Chars:  []byte{']'},
				Ranges: []charRange{
					{
						From: 'a',
						To:   'Z',
					},
				},
			},
			wantConsumed: 7,
		},
		"dont parse if closing not found 1": {
			givenStr: "[",
		},
		"dont parse if closing not found 2": {
			givenStr: "[a",
		},
		"dont parse if closing not found 3": {
			givenStr: "[abc",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// when
			gotBracket, gotConsumed := parseBracket(tt.givenStr)

			// then
			if d := cmp.Diff(tt.wantBracket, gotBracket); d != "" {
				t.Errorf("diff (-want +got):\n%s", d)
			}

			if d := cmp.Diff(tt.wantConsumed, gotConsumed); d != "" {
				t.Errorf("consumed diff (-want +got):\n%s", d)
			}
		})
	}
}

func TestParseQuantifier(t *testing.T) {
	tests := map[string]struct {
		givenStr      string
		wantQuantifer *quantifier
		wantConsumed  int
	}{
		"dont parse if no Quantifier Char": {
			givenStr: "}",
		},
		"happy +": {
			givenStr: "+abc",
			wantQuantifer: &quantifier{
				Min: 1,
				Max: math.MaxInt,
			},
			wantConsumed: 1,
		},
		"happy *": {
			givenStr: "*abc",
			wantQuantifer: &quantifier{
				Min: 0,
				Max: math.MaxInt,
			},
			wantConsumed: 1,
		},
		"happy ?": {
			givenStr: "?abc",
			wantQuantifer: &quantifier{
				Min: 0,
				Max: 1,
			},
			wantConsumed: 1,
		},
		"happy {m,n}": {
			givenStr: "{33,34}abc",
			wantQuantifer: &quantifier{
				Min: 33,
				Max: 34,
			},
			wantConsumed: 7,
		},
		"dont parse if not a number {m,n}": {
			givenStr:      "{33,34a}abc",
			wantQuantifer: nil,
		},
		"dont parse if too few numbers {m,n} 1": {
			givenStr: "{33,}abc",
		},
		"dont parse if too few numbers {m,n} 2": {
			givenStr: "{33}abc",
		},
		"dont parse if too few numbers {m,n} 3": {
			givenStr: "{,}abc",
		},
		"dont parse if no closing } numbers {m,n} 1": {
			givenStr: "{33,34",
		},
		"dont parse if no closing } numbers {m,n} 2": {
			givenStr: "{33,34[234]}",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// when
			gotQuantifier, gotConsumed := parseQuantifier(tt.givenStr)

			// then
			if d := cmp.Diff(tt.wantQuantifer, gotQuantifier); d != "" {
				t.Errorf("diff (-want +got):\n%s", d)
			}

			if d := cmp.Diff(tt.wantConsumed, gotConsumed); d != "" {
				t.Errorf("consumed diff (-want +got):\n%s", d)
			}
		})
	}
}
