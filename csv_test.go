package main

import (
	"strings"
	"testing"
)

func TestImportCSV(t *testing.T) {
	db := newTestDB(t)
	data := "Company,reported,Role Title,Date Reviewed,Status\n" +
		"Acme,Yes,Director,11/19/2026,Applied\n" +
		",,,,\n" +
		"RESUME REVAMP,,,,\n" +
		"Acme,Yes,Director,11/19/2026,Applied\n"

	added, skipped, err := ImportCSV(db, strings.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if added != 1 || skipped != 3 {
		t.Fatalf("added=%d skipped=%d, want 1 and 3", added, skipped)
	}
	apps, err := ListApplications(db)
	if err != nil {
		t.Fatal(err)
	}
	if apps[0].DateReviewed != "2026-11-19" {
		t.Errorf("date = %q, want 2026-11-19", apps[0].DateReviewed)
	}

	added, skipped, err = ImportCSV(db, strings.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if added != 0 || skipped != 4 {
		t.Errorf("re-run added=%d skipped=%d, want 0 and 4", added, skipped)
	}
}
