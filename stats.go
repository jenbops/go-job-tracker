package main

import (
	"database/sql"
	"fmt"
)

// Count is one bar in a chart.
type Count struct {
	Label string
	N     int
	Pct   int // bar width from 3 to 100, relative to the biggest bar in its chart
}

// Stats holds everything the stats page shows.
type Stats struct {
	Total    int
	ByStatus []Count
	BySource []Count
	ByWeek   []Count
}

// GetStats counts applications by status, source and week.
func GetStats(db *sql.DB) (Stats, error) {
	var s Stats
	if err := db.QueryRow(`SELECT COUNT(*) FROM applications`).Scan(&s.Total); err != nil {
		return s, fmt.Errorf("count applications: %w", err)
	}

	var err error
	s.ByStatus, err = countBy(db, `
		SELECT CASE WHEN status = '' THEN '(none)' ELSE status END, COUNT(*)
		FROM applications GROUP BY 1 ORDER BY COUNT(*) DESC, 1`)
	if err != nil {
		return s, err
	}
	s.BySource, err = countBy(db, `
		SELECT CASE WHEN source = '' THEN '(none)' ELSE source END, COUNT(*)
		FROM applications GROUP BY 1 ORDER BY COUNT(*) DESC, 1 LIMIT 10`)
	if err != nil {
		return s, err
	}
	// date(d, '-6 days', 'weekday 1') is the Monday on or before d.
	s.ByWeek, err = countBy(db, `
		SELECT date(date_reviewed, '-6 days', 'weekday 1') AS week, COUNT(*)
		FROM applications
		WHERE date(date_reviewed) IS NOT NULL
		GROUP BY week ORDER BY week DESC LIMIT 12`)
	if err != nil {
		return s, err
	}
	for i := range s.ByWeek {
		s.ByWeek[i].Label = "Week of " + displayDate(s.ByWeek[i].Label)
	}
	return s, nil
}

// countBy runs a query that returns (label, count) rows and sizes the bars.
func countBy(db *sql.DB, query string) ([]Count, error) {
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("stats query: %w", err)
	}
	defer rows.Close()

	var out []Count
	biggest := 0
	for rows.Next() {
		var c Count
		if err := rows.Scan(&c.Label, &c.N); err != nil {
			return nil, fmt.Errorf("read stats row: %w", err)
		}
		if c.N > biggest {
			biggest = c.N
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		out[i].Pct = out[i].N * 100 / biggest
		if out[i].Pct < 3 {
			out[i].Pct = 3 // keep tiny bars visible
		}
	}
	return out, nil
}
