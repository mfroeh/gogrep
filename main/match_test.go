package main

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// not really a unit test, relies on correct parsing
func TestMatch(t *testing.T) {
	tests := map[string]struct {
		givenStrings []string
		givenRe      string
	}{
		"happy banana": {
			givenRe:      ".*ba.*",
			givenStrings: []string{"banana"},
		},
		"happy complex": {
			givenRe:      "([A-Za-z]+|[0-9]{3,5})([_.-][^0-9 ]?)*([A-Za-z0-9_]{2,2}|[0-9]+)",
			givenStrings: []string{"my_value.X_Final99", "my_value.X_Final99999999999999999"},
		},
		"(ab|a)c": {
			givenRe:      "(ab|a)c",
			givenStrings: []string{"abc", "ac"},
		},
		"happy complex 2": {
			givenRe: "(ID|REF)(_?(ALPHA|BETA|[0-9]{2,4}))([_.-][A-Za-z]{3,3}|[0-9]+)*X{1,2}",
			givenStrings: []string{
				"ID_ALPHA_abcX",
				"ID_BETA_xyz.5678_ABCX",
			},
		},
		"happy complex 3": {
			givenRe: `[A-Z][a-z]*(/[^0-9_.-]+|\.[0-9]+)*[A-Z]?`,
			givenStrings: []string{
				"Root",
				"MyPath.123/segment_xyz/another.999End",
				"Folder/sub/item.123.456",
				"File/name.123",
			},
		},
		"happy complex 4": {
			givenRe: `[0-9]{2}:[0-9]{2}:[0-9]{2}(_WARN|_INFO|_ERROR)? ([A-Za-z ]+)?(\[ID:[0-9]+\]|\[MSG:[^]]+\])?`,
			givenStrings: []string{
				"12:00:00_WARN Another message [ID:123]",
				"01:02:03_INFO Detail here [MSG:Hello World]",
			},
		},
		"ere_complex_id_tag": {
			givenRe: "(ID|REF)(_?(ALPHA|BETA|[0-9]{2,4}))([_.-][A-Za-z]{3}|[0-9]+)*X{1,2}",
			givenStrings: []string{
				"IDALPHAXX",                // Basic, no underscore, no repeated group
				"REF_BETA_xyzX",            // Underscore, one repeated group with letters
				"ID1234.5678X",             // Digits in inner choice, repeated group with dot and digits
				"REF_ALPHA_abc.12_DEF.34X", // Multiple repeated groups of different types
				"ID12XX",                   // Shortest valid for ID with 2 digits, X repeated
				"REF_BETA_ghi_7890X",       // Multiple underscore separated groups
			},
		},
		"ere_complex_path_segment": {
			givenRe: `[A-Z][a-z]*(/[^0-9_.-]+|\.[0-9]+)*[A-Z]?`,
			givenStrings: []string{
				"Root",                                  // Basic, no optional segments
				"Folder/sub/item.123.456",               // Mix of slash and dot segments
				"MyPath.123/segment_xyz/another.999End", // Complex path with trailing capital
				"A/b.1/c.2D",                            // More compact segments
				"Name.1",                                // Basic with dot segment
				"Name/Segment",                          // Basic with slash segment
				"N",                                     // Shortest valid string
			},
		},
		"ere_log_line_parser": {
			givenRe: `[0-9]{2}:[0-9]{2}:[0-9]{2}(_WARN|_INFO|_ERROR)? ([A-Za-z ]+)?(\[ID:[0-9]+\]|\[MSG:[^]]+\])?`,
			givenStrings: []string{
				"10:00:00",                               // Basic timestamp only
				"12:34:56_INFO My message",               // Timestamp, level, message
				"01:02:03_ERROR Critical Error [ID:999]", // All parts present
				"23:00:00 [MSG:Data here]",               // Timestamp, no level/message, structured part with non-bracket content
				"05:05:05 Text only",                     // Timestamp, message only
				"00:00:00_WARN",                          // Timestamp, level only
				"11:11:11_INFO",                          // Timestamp, level only (another case)
				"09:09:09 [ID:1]",                        // Timestamp, ID only (min ID)
				"08:08:08 [MSG:Hello there, Gemini!]",    // Timestamp, MSG with more complex content
			},
		},
		"ere_tricky_greedy_star": {
			givenRe: `a.*b(c)?`,
			givenStrings: []string{
				"ab",        // Basic
				"abc",       // Optional 'c' present
				"axb",       // '.*' consumes 'x'
				"axyzb",     // '.*' consumes 'xyz'
				"axbyc",     // '.*' consumes 'xby', finding the last 'b'
				"a.b",       // '.' matches '.'
				"a.bc",      // '.' matches '.', 'c' present
				"a_X_Y_b_c", // Longer string, `.*` consumes all until last `b`
			},
		},
		"ere_tricky_alternation_priority": {
			givenRe: `(aa|a)b+`,
			givenStrings: []string{
				"aab",    // `aa` takes precedence
				"baaabb", // `aa` takes precedence
				"ab",     // `a` is chosen
				"abbb",   // `a` is chosen
				"aaabbbb",
			},
		},
		"ere_tricky_negated_star_optional": {
			givenRe: `x([^y]*)y?`,
			givenStrings: []string{
				"x",       // Shortest valid (no `[^y]*`, no `y?`)
				"xy",      // Shortest with `y`
				"xabc",    // `[^y]*` consumes `abc`
				"xabcy",   // `[^y]*` consumes `abc`, `y?` consumes `y`
				"xzzzz",   // `[^y]*` consumes all `z`
				"x_1_2_3", // Non-'y' characters
				"x.!",     // Non-'y' special characters
			},
		},
		"ere_tricky_literal_dot": {
			givenRe: `([.]|[a-z])\.?`,
			givenStrings: []string{
				".",  // Matches `[.]`
				"a",  // Matches `[a-z]`
				"a.", // Matches `[a-z]` then `\.?`
				"..", // Matches `[.]` then `\.?`
				"z.", // Matches `[a-z]` then `\.?`
				"y",  // Matches `[a-z]`
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// when
			type combination struct {
				Re    string
				Str   string
				Match bool
				Err   error
			}

			gotResults := make([]combination, len(tt.givenStrings))
			for i, s := range tt.givenStrings {
				gotMatch, gotErr := Match(tt.givenRe, s)
				if gotErr != nil {
					t.Fatalf("our Match: %v", gotErr)
				}
				gotResults[i] = combination{tt.givenRe, s, gotMatch, gotErr}
			}

			wantResults := make([]combination, len(tt.givenStrings))
			for i, s := range tt.givenStrings {
				wantResult, err := regexp.Match(tt.givenRe, []byte(s))
				if err != nil {
					t.Fatalf("regexp.Match: %v", err)
				}
				wantResults[i] = combination{tt.givenRe, s, wantResult, err}
			}

			// then
			if d := cmp.Diff(wantResults, gotResults); d != "" {
				t.Errorf("got diff (-want +got):\n%s", d)
			}
			fmt.Println(gotResults)
		})
	}
}
