// Package distance implements edit-distance helpers for typosquat detection.
package distance

import "unicode"

// Levenshtein returns the edit distance between a and b.
func Levenshtein(a, b string) int {
	if a == b {
		return 0
	}
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	// Ensure a is the shorter string to save memory.
	if la > lb {
		a, b = b, a
		la, lb = lb, la
	}
	prev := make([]int, la+1)
	curr := make([]int, la+1)
	for i := 0; i <= la; i++ {
		prev[i] = i
	}
	for j := 1; j <= lb; j++ {
		curr[0] = j
		bj := b[j-1]
		for i := 1; i <= la; i++ {
			cost := 1
			if a[i-1] == bj {
				cost = 0
			}
			del := prev[i] + 1
			ins := curr[i-1] + 1
			sub := prev[i-1] + cost
			curr[i] = min3(del, ins, sub)
		}
		prev, curr = curr, prev
	}
	return prev[la]
}

func min3(a, b, c int) int {
	if a <= b && a <= c {
		return a
	}
	if b <= c {
		return b
	}
	return c
}

// StripSeparators removes '-', '_', and '.' for separator-insensitive compare.
func StripSeparators(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if r == '-' || r == '_' || r == '.' {
			continue
		}
		out = append(out, unicode.ToLower(r))
	}
	return string(out)
}

// Distance returns the minimum of raw Levenshtein and separator-stripped Levenshtein.
// Stripped distance of 0 with unequal raw names counts as 1 (hyphen/underscore swap).
func Distance(a, b string) int {
	raw := Levenshtein(a, b)
	sa, sb := StripSeparators(a), StripSeparators(b)
	if sa == sb {
		if a == b {
			return 0
		}
		return 1
	}
	stripped := Levenshtein(sa, sb)
	if stripped < raw {
		return stripped
	}
	return raw
}
