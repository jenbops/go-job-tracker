package main

import (
	"database/sql"
	"regexp"
	"strings"
	"unicode"
)

// parenRe matches text in parentheses, like "(Remote)".
var parenRe = regexp.MustCompile(`\([^)]*\)`)

// suffixes are trailing words ignored when comparing company names.
var suffixes = map[string]bool{
	"inc": true, "llc": true, "ltd": true, "corp": true, "ai": true, "io": true,
}

// normalizeCompany turns "Horizon3 AI, Inc." into "horizon3".
func normalizeCompany(name string) string {
	name = strings.ToLower(name)
	name = parenRe.ReplaceAllString(name, "")
	name = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return ' '
	}, name)
	words := strings.Fields(name)
	for len(words) > 1 && suffixes[words[len(words)-1]] {
		words = words[:len(words)-1]
	}
	return strings.Join(words, "")
}

// sameCompany compares two already-normalized names.
func sameCompany(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	if a == b {
		return true
	}
	short, long := a, b
	if len(short) > len(long) {
		short, long = long, short
	}
	return len(short) >= 5 && strings.HasPrefix(long, short)
}

// Match is an existing row that looks like the one being checked.
type Match struct {
	App      Application
	SameRole bool
}

// FindDuplicates returns existing rows for the same company.
func FindDuplicates(db *sql.DB, company, role string) ([]Match, error) {
	apps, err := ListApplications(db)
	if err != nil {
		return nil, err
	}
	want := normalizeCompany(company)
	var matches []Match
	for _, a := range apps {
		if sameCompany(want, normalizeCompany(a.Company)) {
			same := strings.EqualFold(strings.TrimSpace(a.RoleTitle), strings.TrimSpace(role))
			matches = append(matches, Match{App: a, SameRole: same})
		}
	}
	return matches, nil
}
