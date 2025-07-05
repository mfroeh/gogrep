package regex

// missing and I want to add:
// build a small GREP tool on top of this library
// multiline support (in particular with $)
// potentially: POSIX character sets in addition to perl ones
// potentially: unicode support
// potentially: more than just ERE support, e.g. non-greedy (lazy) quantifier variants like .+?
// potentially: look ahead/look behind

import (
	"fmt"
	"strings"
	"unicode"
)

type Regex struct {
	root        *node
	strictStart bool
	strictEnd   bool
}

type Submatch struct {
	Offset int
	Str    string
}

func Compile(re string) (Regex, error) {
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

	root, err := parseGroup("("+re+")", 0)
	if err != nil {
		return Regex{}, fmt.Errorf("failed to construct regex from %q: %w", "("+re+")", err)
	}
	return Regex{
		root:        root,
		strictStart: strictStart,
		strictEnd:   strictEnd,
	}, nil
}

// FindAllSubmatches finds up to maxCount submatches of the pattern in the given string
// To return all submatches pass a maxCount of -1
func (re Regex) FindAllSubmatches(s string, maxCount int) [][]Submatch {
	var allSubmatches [][]Submatch
	for i := 0; i < len(s); i += 1 {
		if maxCount != -1 && len(allSubmatches) >= maxCount {
			return allSubmatches
		}
		if re.strictStart && (i != 0 && (s[i-1] != '\n')) {
			continue
		}

		_, match := re.root.match(s[i:], 0, 0)
		if match {
			var submatches []Submatch
			re.root.collectSubmatches(s[i:], &submatches)
			for j := range submatches {
				submatches[j].Offset += i
			}
			rootMatchStr := submatches[0].Str
			if !re.strictEnd ||
				// end of input string
				len(rootMatchStr) == len(s[i:]) ||
				// newline right after
				(len(rootMatchStr)+1 < len(s) && s[len(rootMatchStr)+1] == '\n') {
				allSubmatches = append(allSubmatches, submatches)
				i += len(rootMatchStr) - 1
			}
		}
	}
	return allSubmatches
}

func (re Regex) FindSubmatch(s string) []Submatch {
	submatch := re.FindAllSubmatches(s, 1)
	if len(submatch) < 1 {
		return nil
	}
	return submatch[0]
}

func (re Regex) Match(s string) bool {
	return len(re.FindSubmatch(s)) > 0
}

func (re Regex) Replace(s string, with string) string {
	submatches := re.FindSubmatch(s)
	out := strings.Builder{}
	for i := 0; i < len(with); i++ {
		if with[i] == '$' && i+1 < len(with) && unicode.IsDigit(rune(with[i+1])) {
			num := 0
			for j := i + 1; j < len(with) && unicode.IsDigit(rune(with[j])); j++ {
				num *= 10
				num += int(with[j] - '0')
				i++
			}

			if num < len(submatches) {
				out.WriteString(submatches[num].Str)
			}
		} else {
			out.WriteByte(with[i])
		}
	}
	return out.String()
}
