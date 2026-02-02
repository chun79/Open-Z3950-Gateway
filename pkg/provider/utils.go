package provider

import (
	"regexp"
	"strings"
)

// isbnPrefixRegex matches common ISBN prefixes like "ISBN-13:", "ISBN-10:", "ISBN:".
// It is case-insensitive and handles optional spacing around the colon.
var isbnPrefixRegex = regexp.MustCompile(`(?i)^(isbn-13|isbn-10|isbn)\s*:\s*`)

// isbnCleanRegex matches any character that is not a digit or the letter 'X' (case-insensitive).
var isbnCleanRegex = regexp.MustCompile(`[^0-9xX]`)

// CleanISBN sanitizes a raw string to extract a pure ISBN number.
// It performs a two-step process:
// 1. Removes common prefixes (e.g., "ISBN-13:").
// 2. Removes all remaining non-essential characters (like hyphens and spaces).
func CleanISBN(raw string) string {
	// Step 1: Remove the known prefixes.
	s := isbnPrefixRegex.ReplaceAllString(raw, "")

	// Step 2: Remove all non-digit/non-X characters from the result.
	s = isbnCleanRegex.ReplaceAllString(s, "")

	return strings.TrimSpace(s)
}