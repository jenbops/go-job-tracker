package main

import (
	"database/sql"
	"path/filepath"
	"testing"
)

func TestNormalizeCompany(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"Horizon3 AI", "horizon3"},
		{"Horizon3.ai", "horizon3"},
		{"Horizon3 AI, Inc.", "horizon3"},
		{"Acme (Remote)", "acme"},
		{"  Stripe  ", "stripe"},
		{"AI", "ai"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := normalizeCompany(tt.in); got != tt.want {
				t.Errorf("normalizeCompany(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestSameCompany(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"horizon3", "horizon3", true},
		{"horizon3", "horizon3labs", true},
		{"acme", "acmecorp", false},
		{"apple", "pineapple", false},
		{"", "stripe", false},
	}
	for _, tt := range tests {
		if got := sameCompany(tt.a, tt.b); got != tt.want {
			t.Errorf("sameCompany(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

// newTestDB makes a fresh database in a temp folder that Go deletes afterward.
func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := OpenStore(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestAddAndList(t *testing.T) {
	db := newTestDB(t)
	id, err := AddApplication(db, Application{
		Company: "Acme", RoleTitle: "Director", Reported: "Yes", Status: "Applied",
	})
	if err != nil {
		t.Fatal(err)
	}
	apps, err := ListApplications(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 1 || apps[0].ID != id || apps[0].Company != "Acme" {
		t.Fatalf("unexpected rows: %+v", apps)
	}
}

func TestFindDuplicates(t *testing.T) {
	db := newTestDB(t)
	_, err := AddApplication(db, Application{
		Company: "Horizon3 AI", RoleTitle: "Engineering Manager", Reported: "Yes", Status: "Applied",
	})
	if err != nil {
		t.Fatal(err)
	}

	matches, err := FindDuplicates(db, "Horizon3.ai, Inc.", "engineering manager")
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 || !matches[0].SameRole {
		t.Fatalf("want 1 match with same role, got %+v", matches)
	}

	matches, err = FindDuplicates(db, "Horizon3 AI", "Product Manager")
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 || matches[0].SameRole {
		t.Fatalf("want 1 match with a different role, got %+v", matches)
	}

	matches, err = FindDuplicates(db, "Other Corp", "Anything")
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("want no matches, got %+v", matches)
	}
}
