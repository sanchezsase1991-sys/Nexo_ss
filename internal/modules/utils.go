package modules

import "strings"

func clamp(v, min, max float64) float64 {
	if v < min { return min }
	if v > max { return max }
	return v
}

func clampInt(v, min, max int) int {
	if v < min { return min }
	if v > max { return max }
	return v
}

func containsTagStr(tags []string, tag string) bool {
	for _, t := range tags { if t == tag { return true } }
	return false
}

func containsStr(slice []string, item string) bool {
	for _, s := range slice { if s == item { return true } }
	return false
}

func countTrue(bools ...bool) int {
	c := 0
	for _, b := range bools { if b { c++ } }
	return c
}

func containsSubstringStr(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

func truncateStr(s string, maxLen int) string {
	if len(s) > maxLen { return s[:maxLen-3] + "..." }
	return s
}

func clampF64(v, lo, hi float64) float64 {
	if v < lo { return lo }
	if v > hi { return hi }
	return v
}

func isNumeric(s string) bool {
	for _, c := range s { if c < '0' || c > '9' { return false } }
	return len(s) > 0
}
