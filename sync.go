package main

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
)

// SyncReport describes what SyncCSV found when it compared a sheet export
// with the database.
type SyncReport struct {
	New     []Application // in the sheet but not the database
	Changed []string      // one readable line per row whose fields differ
	Same    int           // rows that already match
	OnlyDB  []Application // in the database but not the sheet (never deleted)
}

// sheetKey identifies a row. Two rows are the same application when the
// company, role and date reviewed all match.
//
// A struct with only comparable fields (strings here) can be used as a map
// key in Go, which is why this works without building a combined string.
type sheetKey struct {
	company, role, date string
}

// SyncCSV compares a Google Sheet export with the database. New rows are
// added and changed rows are updated to match the sheet (the sheet wins),
// unless dryRun is true. Rows found only in the database are reported but
// never deleted.
func SyncCSV(db *sql.DB, r io.Reader, dryRun bool) (SyncReport, error) {
	var rep SyncReport

	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1 // allow rows with different column counts

	header, err := cr.Read()
	if err != nil {
		return rep, fmt.Errorf("read header: %w", err)
	}
	index := headerIndex(header)
	for _, need := range []string{"company", "role title"} {
		if _, ok := index[need]; !ok {
			return rep, fmt.Errorf("header has no %q column; found %q", need, header)
		}
	}
	get := columnGetter(index)

	existing, err := ListApplications(db)
	if err != nil {
		return rep, err
	}
	byKey := make(map[sheetKey]Application)
	for _, a := range existing {
		byKey[sheetKey{a.Company, a.RoleTitle, a.DateReviewed}] = a
	}

	seen := make(map[sheetKey]bool)
	for {
		rec, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return rep, fmt.Errorf("read row: %w", err)
		}
		in := Application{
			Company:        get(rec, "Company"),
			Reported:       get(rec, "reported"),
			RoleTitle:      get(rec, "Role Title"),
			Location:       get(rec, "Location"),
			CompRange:      get(rec, "Comp Range"),
			Source:         get(rec, "Platform/Source"),
			DateReviewed:   normalizeDate(get(rec, "Date Reviewed")),
			Status:         get(rec, "Status"),
			Fit:            get(rec, "Fit Assessment"),
			Notes:          get(rec, "Key Gaps / Notes"),
			TailoredResume: get(rec, "Materials Tailored?"),
			NextAction:     get(rec, "Next Action"),
		}
		if in.Company == "" || in.RoleTitle == "" {
			continue // blank rows and divider rows like "RESUME REVAMP"
		}
		k := sheetKey{in.Company, in.RoleTitle, in.DateReviewed}
		if seen[k] {
			continue // exact duplicate row inside the sheet
		}
		seen[k] = true

		cur, found := byKey[k]
		if !found {
			rep.New = append(rep.New, in)
			if !dryRun {
				if _, err := AddApplication(db, in); err != nil {
					return rep, err
				}
			}
			continue
		}

		diffs := diffFields(cur, in)
		if len(diffs) == 0 {
			rep.Same++
			continue
		}
		rep.Changed = append(rep.Changed, fmt.Sprintf("#%d %s, %s: %s",
			cur.ID, cur.Company, cur.RoleTitle, strings.Join(diffs, "; ")))
		if !dryRun {
			in.ID = cur.ID
			if _, err := UpdateApplication(db, in); err != nil {
				return rep, err
			}
		}
	}

	// Walk the original list, not the map, so the report keeps id order.
	for _, a := range existing {
		if !seen[sheetKey{a.Company, a.RoleTitle, a.DateReviewed}] {
			rep.OnlyDB = append(rep.OnlyDB, a)
		}
	}
	return rep, nil
}

// diffFields lists the fields that differ between the database row and the
// sheet row. Company, role and date are the key, so they always match.
func diffFields(cur, in Application) []string {
	var diffs []string
	cmp := func(name, old, new string) {
		if old != new {
			diffs = append(diffs, fmt.Sprintf("%s %q -> %q", name, old, new))
		}
	}
	cmp("reported", cur.Reported, in.Reported)
	cmp("location", cur.Location, in.Location)
	cmp("comp", cur.CompRange, in.CompRange)
	cmp("source", cur.Source, in.Source)
	cmp("status", cur.Status, in.Status)
	cmp("fit", cur.Fit, in.Fit)
	cmp("notes", cur.Notes, in.Notes)
	cmp("materials", cur.TailoredResume, in.TailoredResume)
	cmp("next", cur.NextAction, in.NextAction)
	return diffs
}
