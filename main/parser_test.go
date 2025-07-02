package main

import (
	"github.com/google/go-cmp/cmp"
	"math"
	"testing"
)

func TestParse(t *testing.T) {
	tests := map[string]struct {
		givenStr     string
		wantNode     astNode
		wantConsumed int
	}{
		"(^.*a.*$)": {
			givenStr: "(^.*a.*$)",
			wantNode: &group{
				Children: []astNode{
					char{Char: '^'},
					char{Char: '.', Quantifier: quantifierMul},
					char{Char: 'a'},
					char{Char: '.', Quantifier: quantifierMul},
					char{Char: '$'},
				},
			},
			wantConsumed: 9,
		},
		"(^[^x]*x+$)": {
			givenStr: "(^[^x]*x+$)",
			wantNode: &group{
				Children: []astNode{
					char{Char: '^'},
					&bracket{
						Negate:     true,
						Chars:      []byte{'x'},
						Quantifier: quantifierMul,
					},
					char{Char: 'x', Quantifier: quantifierPlus},
					char{Char: '$'},
				},
			},
			wantConsumed: 11,
		},
		"((ab|a)c)": {
			givenStr: "((ab|a)|c)",
			wantNode: &group{
				Children: []astNode{
					&choices{
						Choices: [][]astNode{
							{
								&group{
									Children: []astNode{
										&choices{
											Choices: [][]astNode{
												{
													char{Char: 'a'},
													char{Char: 'b'},
												},
												{
													char{Char: 'a'},
												},
											},
										},
									},
								},
							},
							{
								char{Char: 'c'},
							},
						},
					},
				},
			},
			wantConsumed: 10,
		},
		"(ab|a)c": {
			givenStr: "((ab|a)c)",
			wantNode: &group{
				Children: []astNode{
					&group{
						Children: []astNode{
							&choices{
								Choices: [][]astNode{
									{
										char{Char: 'a'},
										char{Char: 'b'},
									},
									{
										char{Char: 'a'},
									},
								},
							},
						},
					},
					char{Char: 'c'},
				},
			},
			wantConsumed: 9,
		},
		"complex": {
			givenStr: "(^([A-Za-z]+|[0-9]{3,5})([_.-][^0-9 ]?)*([A-Za-z0-9_]{2,2}|[0-9]+)$)",
			wantNode: &group{
				Children: []astNode{
					char{Char: '^'},
					&group{
						Children: []astNode{
							&choices{
								Choices: [][]astNode{
									{
										&bracket{
											Negate: false,
											Ranges: []charRange{
												{From: 'A', To: 'Z'},
												{From: 'a', To: 'z'},
											},
											Quantifier: quantifierPlus,
										},
									},
									{
										&bracket{
											Negate: false,
											Ranges: []charRange{
												{From: '0', To: '9'},
											},
											Quantifier: &quantifier{Min: 3, Max: 5},
										},
									},
								},
							},
						},
					},
					&group{
						Quantifier: quantifierMul,
						Children: []astNode{
							&bracket{
								Chars: []byte{'_', '.', '-'},
							},
							&bracket{
								Negate:     true,
								Chars:      []byte{' '},
								Ranges:     []charRange{{From: '0', To: '9'}},
								Quantifier: quantifierOptional,
							},
						},
					},
					&group{
						Children: []astNode{
							&choices{
								Choices: [][]astNode{
									{
										&bracket{
											Chars: []byte{'_'},
											Ranges: []charRange{
												{From: 'A', To: 'Z'},
												{From: 'a', To: 'z'},
												{From: '0', To: '9'},
											},
											Quantifier: &quantifier{Min: 2, Max: 2},
										},
									},
									{
										&bracket{
											Ranges:     []charRange{{From: '0', To: '9'}},
											Quantifier: quantifierPlus,
										},
									},
								},
							},
						},
					},
					char{Char: '$'},
				},
			},
			wantConsumed: 68,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// when
			gotNode, gotConsumed := parseGroup(tt.givenStr)

			// then
			if d := cmp.Diff(tt.wantNode, gotNode); d != "" {
				t.Errorf("diff (-want +got):\n%s", d)
			}

			if d := cmp.Diff(tt.wantConsumed, gotConsumed); d != "" {
				t.Errorf("consumed diff (-want +got):\n%s", d)
			}
		})
	}
}

func TestParseChoices(t *testing.T) {
	tests := map[string]struct {
		givenStr     string
		wantChoices  *choices
		wantConsumed int
	}{
		"happy two literal Choices": {
			givenStr: "abc|def",
			wantChoices: &choices{
				Choices: [][]astNode{
					{
						char{Char: 'a'},
						char{Char: 'b'},
						char{Char: 'c'},
					},
					{
						char{Char: 'd'},
						char{Char: 'e'},
						char{Char: 'f'},
					},
				},
			},
			wantConsumed: 7,
		},
		"happy three choices": {
			givenStr: "abc|def|a*b",
			wantChoices: &choices{
				Choices: [][]astNode{
					{
						char{Char: 'a'},
						char{Char: 'b'},
						char{Char: 'c'},
					},
					{
						char{Char: 'd'},
						char{Char: 'e'},
						char{Char: 'f'},
					},
					{
						char{Char: 'a', Quantifier: &quantifier{Min: 0, Max: math.MaxInt}},
						char{Char: 'b'},
					},
				},
			},
			wantConsumed: 11,
		},
		"happy nested choices": {
			givenStr: "(a|c)?|a*b",
			wantChoices: &choices{
				Choices: [][]astNode{
					{
						&group{
							Children: []astNode{&choices{
								Choices: [][]astNode{
									{
										char{Char: 'a'},
									},
									{
										char{Char: 'c'},
									},
								},
							},
							},
							Quantifier: &quantifier{Min: 0, Max: 1},
						},
					},
					{
						char{Char: 'a', Quantifier: &quantifier{Min: 0, Max: math.MaxInt}},
						char{Char: 'b'},
					},
				},
			},
			wantConsumed: 10,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// when
			gotChoices, gotConsumed := parseChoices(tt.givenStr)

			// then
			if d := cmp.Diff(tt.wantChoices, gotChoices); d != "" {
				t.Errorf("diff (-want +got):\n%s", d)
			}

			if d := cmp.Diff(tt.wantConsumed, gotConsumed); d != "" {
				t.Errorf("consumed diff (-want +got):\n%s", d)
			}
		})
	}
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
		"happy {m} 1": {
			givenStr: "{33}abc",
			wantQuantifer: &quantifier{
				Min: 33,
				Max: 33,
			},
			wantConsumed: 4,
		},
		"happy {m} 2": {
			givenStr: "{3}abc",
			wantQuantifer: &quantifier{
				Min: 3,
				Max: 3,
			},
			wantConsumed: 3,
		},
		"dont parse if not a number {m,n}": {
			givenStr:      "{33,34a}abc",
			wantQuantifer: nil,
		},
		"dont parse if too few numbers {m,n} 1": {
			givenStr: "{33,}abc",
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
