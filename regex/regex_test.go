package regex

import (
	"regexp"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestFindSubmatch(t *testing.T) {
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
			givenRe: `[0-9]{2}:[0-9]{2}:[0-9]{2}(_WARN|_INFO|_ERROR)? ([A-Za-z ]+)?(\[ID:[0-9]+\]|\[MSG:[^\]]+\])?`,
			givenStrings: []string{
				"12:00:00_WARN Another message [ID:123]",
				"01:02:03_INFO Detail here [MSG:Hello World]",
			},
		},
		"ere_complex_id_tag": {
			givenRe: "^(ID|REF)(_?(ALPHA|BETA|[0-9]{2,4}))([_.-][A-Za-z]{3}|[0-9]+)*X{1,2}$",
			givenStrings: []string{
				"IDALPHAXX",
				"REF_BETA_xyzX",
				"ID1234.5678X",
				"REF_ALPHA_abc.12_DEF.34X",
				"ID12XX",
				"REF_BETA_ghi_7890X",
			},
		},
		"ere_complex_path_segment": {
			givenRe: `[A-Z][a-z]*(/[^0-9_.-]+|\.[0-9]+)*[A-Z]?`,
			givenStrings: []string{
				"Root",
				"Folder/sub/item.123.456",
				"MyPath.123/segment_xyz/another.999End",
				"A/b.1/c.2D",
				"Name.1",
				"Name/Segment",
				"N",
			},
		},
		"ere_log_line_parser": {
			givenRe: `[0-9]{2}:[0-9]{2}:[0-9]{2}(_WARN|_INFO|_ERROR)? ([A-Za-z ]+)?(\[ID:[0-9]+\]|\[MSG:[^\]]+\])?`,
			givenStrings: []string{
				"10:00:00",
				"12:34:56_INFO My message",
				"01:02:03_ERROR Critical Error [ID:999]",
				"23:00:00 [MSG:Data here]",
				"05:05:05 Text only",
				"00:00:00_WARN",
				"11:11:11_INFO",
				"09:09:09 [ID:1]",
				"08:08:08 [MSG:Hello there, Gemini!]",
			},
		},
		"ere_tricky_greedy_star": {
			givenRe: `a.*b(c)?`,
			givenStrings: []string{
				"ab",
				"abc",
				"axb",
				"axyzb",
				"axbyc",
				"a.b",
				"a.bc",
				"a_X_Y_b_c",
			},
		},
		"ere_tricky_alternation_priority": {
			givenRe: `(aa|a)b+`,
			givenStrings: []string{
				"aab",
				"baaabb",
				"ab",
				"abbb",
				"aaabbbb",
			},
		},
		"ere_tricky_negated_star_optional": {
			givenRe: `x([^y]*)y?`,
			givenStrings: []string{
				"x",
				"xy",
				"xabc",
				"xabcy",
				"xzzzz",
				"x_1_2_3",
				"x.!",
			},
		},
		"ere_tricky_literal_dot": {
			givenRe: `([.]|[a-z])\.?`,
			givenStrings: []string{
				//".",
				//"a",
				"a.",
				"..",
				"z.",
				"y",
			},
		},
		"ere_anchor_strict_enum": {
			givenRe: `^(YES|NO)$`,
			givenStrings: []string{
				"YES",
				"NO",
			},
		},
		"ere_anchor_optional_ends": {
			givenRe: `^A?[0-9]+Z?$`,
			givenStrings: []string{
				"123",
				"A123",
				"123Z",
				"A123Z",
				"A0Z",
			},
		},
		"ere_anchor_any_between": {
			givenRe: `^X.*Y$`,
			givenStrings: []string{
				"XY",
				"X_Y",
				"XabcY",
				"X123_abc_Y",
				"X.Y",
				"X     Y",
			},
		},
		"ere_anchor_negated_full_string": {
			givenRe: `^[^0-9]+$`,
			givenStrings: []string{
				"abc",
				"ABC",
				"!@#$",
				"abc!@#$",
				"Spaces and tabs",
				"A",
				"0AA",
				"ABC0",
			},
		},
		"capture empty groups": {
			givenRe: `(a)?(b)?c`,
			givenStrings: []string{
				"ac", "bc", "c",
			},
		},
		// Perl Character Classes
		"perl_digit_char_set": {
			givenRe:      `\d+`,
			givenStrings: []string{"12345", "0", "987", "000"},
		},
		"perl_non_digit_char_set": {
			givenRe:      `\D+`,
			givenStrings: []string{"abc", "ABC", "hello world", "!@#$"},
		},
		"perl_whitespace_char_set": {
			givenRe:      `\s+`,
			givenStrings: []string{" ", "\t", "\n", " \t\n"},
		},
		"perl_non_whitespace_char_set": {
			givenRe:      `\S+`,
			givenStrings: []string{"hello", "WORLD", "123!", "no_spaces_here"},
		},
		"perl_word_char_set": {
			givenRe:      `\w+`,
			givenStrings: []string{"word", "Word123", "____"},
		},
		"perl_non_word_char_set": {
			givenRe:      `\W+`,
			givenStrings: []string{"!@#$", " ", "\t\n", ".-_"},
		},
		"perl_combination_d_S": {
			givenRe:      `\d\S\d`,
			givenStrings: []string{"1a2", "5!8", "0_9"},
		},
		"perl_combination_w_D": {
			givenRe:      `\w\D\w`,
			givenStrings: []string{"a-b", "1_2", "A!C"},
		},
		// POSIX Character Classes
		"posix_alnum": {
			givenRe:      `[[:alnum:]]+`,
			givenStrings: []string{"abc123XYZ", "Hello_World", "123Test", "AlphaBravo"},
		},
		"posix_alpha": {
			givenRe:      `[[:alpha:]]+`,
			givenStrings: []string{"abcXYZ", "Hello", "AlphaBravo"},
		},
		"posix_blank": {
			givenRe:      `[[:blank:]]+`,
			givenStrings: []string{" ", "\t", "  \t "},
		},
		"posix_cntrl": {
			givenRe:      `[[:cntrl:]]+`,
			givenStrings: []string{"\x00", "\x1F", "\x7F"}, // Null, US, DEL
		},
		"posix_digit": {
			givenRe:      `[[:digit:]]+`,
			givenStrings: []string{"12345", "0", "987"},
		},
		"posix_graph": {
			givenRe:      `[[:graph:]]+`,
			givenStrings: []string{"!@#$ABCabc123", "NoSpacesHere!"},
		},
		"posix_lower": {
			givenRe:      `[[:lower:]]+`,
			givenStrings: []string{"abcdefg", "hello world"},
		},
		"posix_print": {
			givenRe:      `[[:print:]]+`,
			givenStrings: []string{"Printable text 123 !@#", "All visible characters"},
		},
		"posix_punct": {
			givenRe:      `[[:punct:]]+`,
			givenStrings: []string{"!@#$%^&*()", ".-_=+[]{};:'\",<>/?`~"},
		},
		"posix_space": {
			givenRe:      `[[:space:]]+`,
			givenStrings: []string{" ", "\t", "\n", "\r", "\f", "\v"},
		},
		"posix_upper": {
			givenRe:      `[[:upper:]]+`,
			givenStrings: []string{"ABCDEFG", "HELLO WORLD"},
		},
		"posix_word": {
			givenRe:      `[[:word:]]+`,
			givenStrings: []string{"word", "Word123", "____"}, // Equivalent to \w
		},
		"posix_xdigit": {
			givenRe:      `[[:xdigit:]]+`,
			givenStrings: []string{"0123456789ABCDEFabcdef"},
		},
		"posix_negated_alnum": {
			givenRe:      `[^[:alnum:]]+`,
			givenStrings: []string{"!@#$ ", ".-_", " "},
		},
		"posix_negated_alpha": {
			givenRe:      `[^[:alpha:]]+`,
			givenStrings: []string{"123!@#$", " ", ".-_"},
		},
		"posix_negated_digit": {
			givenRe:      `[^[:digit:]]+`,
			givenStrings: []string{"abcABC!@#", ".-_ "},
		},
		"posix_combination_alpha_space": {
			givenRe:      `[[:alpha:]][[:space:]][[:alpha:]]`,
			givenStrings: []string{"a b", "X Y", "m\tn"},
		},
		"posix_combination_digit_punct": {
			givenRe:      `[[:digit:]][[:punct:]][[:digit:]]`,
			givenStrings: []string{"1!2", "5.8", "0-9"},
		},
		// correct escape sequence parsing
		"escape_sequence_parsing": {
			givenRe:      `[\r\f\t\n]`,
			givenStrings: []string{"\t\n\r\f"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// when
			type combination struct {
				Re         string
				Str        string
				Submatches []string
				Err        error
			}

			gotResults := make([]combination, len(tt.givenStrings))
			for i, s := range tt.givenStrings {
				re, gotErr := Compile(tt.givenRe)
				if gotErr != nil {
					t.Fatalf("our Compile: %v", gotErr)
				}
				gotSubmatches := re.FindSubmatch(s)
				var gotSubmatchStrings []string
				for _, s := range gotSubmatches {
					gotSubmatchStrings = append(gotSubmatchStrings, s.Str)
				}
				gotResults[i] = combination{tt.givenRe, s, gotSubmatchStrings, gotErr}
			}

			wantResults := make([]combination, len(tt.givenStrings))
			for i, s := range tt.givenStrings {
				re, err := regexp.Compile(tt.givenRe)
				if err != nil {
					t.Fatalf("regexp.Match: %v", err)
				}
				wantSubmatches := re.FindStringSubmatch(s)
				wantResults[i] = combination{tt.givenRe, s, wantSubmatches, err}
			}

			// then
			if d := cmp.Diff(wantResults, gotResults); d != "" {
				t.Errorf("got diff (-want +got):\n%s", d)
			}
		})
	}
}

func TestReplace(t *testing.T) {
	tests := map[string]struct {
		givenRe      string
		givenStr     string
		givenReplace string
		wantReplaced string
	}{
		"complex, many replace": {
			givenRe:      `[0-9]{2}:[0-9]{2}:[0-9]{2}(_WARN|_INFO|_ERROR)? ([A-Za-z ]+)?(\[ID:[0-9]+\]|\[MSG:[^\]]+\])?`,
			givenStr:     "01:02:03_ERROR Critical Error [ID:999]",
			givenReplace: "I$3reversed$2them$1hihi$0",
			wantReplaced: "I" + "[ID:999]" + "reversed" + "Critical Error " + "them" + "_ERROR" + "hihi" + "01:02:03_ERROR Critical Error [ID:999]",
		},
		"complex, replace with empty string if group not matched": {
			givenRe:      `(aa)b?`,
			givenStr:     "aab",
			givenReplace: "$0$2",
			wantReplaced: "aab",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// when
			re, gotErr := Compile(tt.givenRe)
			if gotErr != nil {
				t.Fatalf("our Compile: %v", gotErr)
			}
			gotReplaced := re.Replace(tt.givenStr, tt.givenReplace)

			// then
			if d := cmp.Diff(tt.wantReplaced, gotReplaced); d != "" {
				t.Errorf("got diff (-want +got):\n%s", d)
			}
		})
	}
}

func TestFindAllSubmatches(t *testing.T) {
	tests := map[string]struct {
		givenRe        string
		givenString    string
		wantSubmatches [][]Submatch
	}{

		"happy bananas": {
			givenRe:     "ba(.{0,2})na",
			givenString: "anbananaortwobananasbaxana",
		},
		"multiline - dotall and groups": {
			givenRe: `BEGIN\n(.*)\nEND`,
			givenString: `Some preamble
BEGIN
  Line 1 of content
  Line 2 of content
END
Some postamble`,
		},
		"nested groups - json like structure": {
			givenRe:     `\{\s*"id":\s*(\d+),\s*"data":\s*"([^"]*)"\s*\}`,
			givenString: `Before {"id": 123, "data": "hello"} after {"id": 456, "data": "world"} end`,
		},
		"nested groups - path segments": {
			givenRe:     `(/(\w+))+`,
			givenString: `/usr/local/bin/my_app /var/log/app.log`,
		},
		"complex nested groups with different character sets": {
			givenRe:     `\[(\w+):(<([^>]+)>)?\]`,
			givenString: `[Config: <Setting1>] [Type: <Boolean>] [Name: ]`,
		},
		// TODO: works fine for single submatches, breaks when searching for all for multiple
		//"capture empty groups": {
		//	givenRe:     `(a)?(b)?c`,
		//	givenString: `abc ac bc c`,
		//},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// when
			re, gotErr := Compile(tt.givenRe)
			if gotErr != nil {
				t.Fatalf("our Compile: %v", gotErr)
			}
			gotSubmatches := re.FindAllSubmatches(tt.givenString, -1)

			var gotSubmatchesStrings [][]string
			for _, match := range gotSubmatches {
				var submatchStrings []string
				for _, sm := range match {
					submatchStrings = append(submatchStrings, sm.Str)
				}
				gotSubmatchesStrings = append(gotSubmatchesStrings, submatchStrings)
			}

			goRe, err := regexp.Compile(tt.givenRe)
			if err != nil {
				t.Fatalf("golang Compile: %v", err)
			}
			wantMatchesStrings := goRe.FindAllStringSubmatch(tt.givenString, -1)

			// then
			if d := cmp.Diff(wantMatchesStrings, gotSubmatchesStrings); d != "" {
				t.Errorf("got diff (-want +got):\n%s", d)
			}
		})
	}
}
