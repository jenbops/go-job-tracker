package main

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"time"
)

// csvHeaders matches the columns of the Google Sheet.
var csvHeaders = []string{
	"Company", "reported", "Role Title", "Location", "Comp Range",
	"Platform/Source", "Date Reviewed", "Status", "Fit Assessment",
	"Key Gaps / Notes", "Materials Tailored?", "Next Action",
}

// normalizeDate turns m/d/yyyy into yyyy-mm-dd. Anything else is kept as typed.
func normalizeDate(s string) string {
	s = strings.TrimSpace(s)
	for _, layout := range []string{"1/2/2006", "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.Format("2006-01-02")
		}
	}
	return s
}

// displayDate turns yyyy-mm-dd into m/d/yyyy for export.
func displayDate(s string) string {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return s
	}
	return t.Format("1/2/2006")
}

// ImportCSV reads rows from r and adds the ones not already present.
// It returns how many rows were added and how many were skipped.
func ImportCSV(db *sql.DB, r io.Reader) (int, int, error) {
	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1 // allow rows with different column counts

	header, err := cr.Read()
	if err != nil {
		return 0, 0, fmt.Errorf("read header: %w", err)
	}
	index := make(map[string]int)
	for i, h := range header {
		h = strings.TrimPrefix(h, "\ufeff") // some exports start with an invisible marker
		index[strings.ToLower(strings.TrimSpace(h))] = i
	}
	for _, need := range []string{"company", "role title"} {
		if _, ok := index[need]; !ok {
			return 0, 0, fmt.Errorf("header has no %q column; found %q", need, header)
		}
	}
	get := func(rec []string, name string) string {
		i, ok := index[strings.ToLower(name)]
		if !ok || i >= len(rec) {
			return ""
		}
		return strings.TrimSpace(rec[i])
	}

	added, skipped := 0, 0
	for {
		rec, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return added, skipped, fmt.Errorf("read row: %w", err)
		}
		a := Application{
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
		if a.Company == "" || a.RoleTitle == "" {
			skipped++ // blank rows and divider rows like "RESUME REVAMP"
			continue
		}
		var n int
		err = db.QueryRow(`SELECT COUNT(*) FROM applications
			WHERE company = ? AND role_title = ? AND date_reviewed = ?`,
			a.Company, a.RoleTitle, a.DateReviewed).Scan(&n)
		if err != nil {
			return added, skipped, err
		}
		if n > 0 {
			skipped++ // already present, so re-running is safe
			continue
		}
		if _, err := AddApplication(db, a); err != nil {
			return added, skipped, err
		}
		added++
	}
	return added, skipped, nil
}

// ExportCSV writes every row to w using the sheet's columns.
func ExportCSV(db *sql.DB, w io.Writer) error {
	apps, err := ListApplications(db)
	if err != nil {
		return err
	}
	cw := csv.NewWriter(w)
	if err := cw.Write(csvHeaders); err != nil {
		return err
	}
	for _, a := range apps {
		row := []string{
			a.Company, a.Reported, a.RoleTitle, a.Location, a.CompRange,
			a.Source, displayDate(a.DateReviewed), a.Status, a.Fit,
			a.Notes, a.TailoredResume, a.NextAction,
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}
