package main

import (
	"strings"
	"testing"
)

func TestSyncCSV(t *testing.T) {
	db := newTestDB(t)
	sample := "Company,reported,Role Title,Date Reviewed,Status\n" +
		"Acme,Yes,Director,11/19/2026,Rejected\n" +
		"Globex,Yes,Manager,11/18/2026,Applied\n"
	if _, _, err := ImportCSV(db, strings.NewReader(sample)); err != nil {
		t.Fatal(err)
	}

	updated := strings.Replace(sample, "Rejected", "Interview", 1) +
		"Initech,No,SRE,11/20/2026,Passed\n" +
		",,,,\n" + // blank row is ignored
		"Initech,No,SRE,11/20/2026,Passed\n" // exact duplicate inside the sheet

	// Dry run reports the differences and changes nothing.
	rep, err := SyncCSV(db, strings.NewReader(updated), true)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.New) != 1 || len(rep.Changed) != 1 || rep.Same != 1 || len(rep.OnlyDB) != 0 {
		t.Fatalf("dry run: %+v", rep)
	}
	if apps, _ := ListApplications(db); len(apps) != 2 {
		t.Fatalf("dry run changed the database: %d rows", len(apps))
	}

	// A real run adds the new row and updates the changed one.
	if _, err := SyncCSV(db, strings.NewReader(updated), false); err != nil {
		t.Fatal(err)
	}
	apps, _ := ListApplications(db)
	if len(apps) != 3 {
		t.Fatalf("rows after sync = %d, want 3", len(apps))
	}
	if apps[0].Status != "Interview" {
		t.Errorf("status = %q, want Interview", apps[0].Status)
	}

	// Running it again finds nothing to do.
	rep, _ = SyncCSV(db, strings.NewReader(updated), false)
	if len(rep.New) != 0 || len(rep.Changed) != 0 || rep.Same != 3 {
		t.Fatalf("second sync is not idempotent: %+v", rep)
	}

	// A row missing from the sheet is reported but kept.
	rep, _ = SyncCSV(db, strings.NewReader(sample), true)
	if len(rep.OnlyDB) != 1 || rep.OnlyDB[0].Company != "Initech" {
		t.Errorf("OnlyDB = %+v, want just Initech", rep.OnlyDB)
	}
	if apps, _ := ListApplications(db); len(apps) != 3 {
		t.Errorf("sync deleted a row: %d left", len(apps))
	}
}
