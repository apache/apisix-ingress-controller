package utils

func TruncateString(s string, max int) string {
	if max > len(s) || max < 0 {
		return s
	}
	return s[:max]
}
