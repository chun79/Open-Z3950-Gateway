package z3950

import (
	"strings"
	"testing"
)

func TestMARCBuildAndParse(t *testing.T) {
	testCases := []struct {
		name      string
		profile   *MARCProfile
		title     string
		author    string
		isbn      string
		publisher string
		year      string
		issn      string
		subject   string
	}{
		{
			name:      "MARC21 Standard",
			profile:   &ProfileMARC21,
			title:     "Golang Programming",
			author:    "Rob Pike",
			isbn:      "9781234567890",
			publisher: "O'Reilly",
			year:      "2024",
			issn:      "1234-5678",
			subject:   "Computer Science",
		},
		{
			name:      "CNMARC Standard",
			profile:   &ProfileCNMARC,
			title:     "红楼梦",
			author:    "曹雪芹",
			isbn:      "9787020002207",
			publisher: "人民文学出版社",
			year:      "1982",
			issn:      "",
			subject:   "Classic Literature",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data := BuildMARC(tc.profile, "001", tc.title, tc.author, tc.isbn, tc.publisher, tc.year, tc.issn, tc.subject)
			if len(data) == 0 {
				t.Fatal("BuildMARC returned empty data")
			}

			rec, err := ParseMARC(data)
			if err != nil {
				t.Fatalf("ParseMARC failed: %v", err)
			}

			parsedTitle := rec.GetTitle(tc.profile)
			if !contains(parsedTitle, tc.title) {
				t.Errorf("Title mismatch: expected %s, got %s", tc.title, parsedTitle)
			}

			parsedISBN := rec.GetISBN(tc.profile)
			if !contains(parsedISBN, tc.isbn) {
				t.Errorf("ISBN mismatch: expected %s, got %s", tc.isbn, parsedISBN)
			}

			if tc.issn != "" {
				parsedISSN := rec.GetISSN(tc.profile)
				if !contains(parsedISSN, tc.issn) {
					t.Errorf("ISSN mismatch: expected %s, got %s", tc.issn, parsedISSN)
				}
			}

			if tc.subject != "" {
				parsedSubject := rec.GetSubject(tc.profile)
				if !contains(parsedSubject, tc.subject) {
					t.Errorf("Subject mismatch: expected %s, got %s", tc.subject, parsedSubject)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}