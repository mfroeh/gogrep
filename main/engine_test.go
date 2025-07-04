package main

import (
	"math"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestMatchInternal(t *testing.T) {
	tests := map[string]struct {
		givenInput string
		givenNode  *node
		wantAdv    int
		wantMatch  bool
	}{
		`happy a?: "b"`: {
			givenInput: "b",
			givenNode: &node{
				state: &charState{char: 'a'},
				mi:    0,
				ma:    1,
				next:  nil,
			},
			wantAdv:   0,
			wantMatch: true,
		},
		`happy a?: "a"`: {
			givenInput: "a",
			givenNode: &node{
				state: &charState{char: 'a'},
				mi:    0,
				ma:    1,
				next:  nil,
			},
			wantAdv:   1,
			wantMatch: true,
		},
		`happy .*ba.*: "banana"`: {
			givenInput: "banana",
			givenNode: &node{
				state: &groupState{
					firstChild: &node{
						state: &charState{
							char: '.',
						},
						mi: 0,
						ma: math.MaxInt,
						next: &node{
							state: &charState{
								char: 'b',
							},
							mi: 1,
							ma: 1,
							next: &node{
								state: &charState{
									char: 'a',
								},
								mi: 1,
								ma: 1,
								next: &node{
									state: &charState{
										char: '.',
									},
									mi: 0,
									ma: math.MaxInt,
								},
							},
						},
					},
				},
				mi: 1,
				ma: 1,
			},
			wantAdv:   6,
			wantMatch: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// when
			gotAdv, gotMatch := tt.givenNode.match(tt.givenInput, 0, 0)

			// then
			if d := cmp.Diff(tt.wantAdv, gotAdv); d != "" {
				t.Errorf("got diff (-want +got):\n%s", d)
			}

			if d := cmp.Diff(tt.wantMatch, gotMatch); d != "" {
				t.Errorf("got diff (-want +got):\n%s", d)
			}
		})
	}
}
