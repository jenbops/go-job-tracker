package main

import (
	"database/sql"
	"errors"
	"fmt"
)

// Application is one row of the applications table.
type Application struct {
	ID             int64
	Company        string
	Reported       string
	RoleTitle      string
	Location       string
	CompRange      string
	Source         string
	DateReviewed   string
	Status         string
	Fit            string
	Notes          string
	TailoredResume string
	NextAction     string
}

// appColumns lists the columns in the order scanApplication expects them.
const appColumns = `id, company, reported, role_title, location, comp_range, source,
	date_reviewed, status, fit, notes, tailored_resume, next_action`

// scanner is satisfied by both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanApplication(s scanner) (Application, error) {
	var a Application
	err := s.Scan(&a.ID, &a.Company, &a.Reported, &a.RoleTitle, &a.Location,
		&a.CompRange, &a.Source, &a.DateReviewed, &a.Status, &a.Fit,
		&a.Notes, &a.TailoredResume, &a.NextAction)
	return a, err
}

// AddApplication inserts one row and returns its new id.
func AddApplication(db *sql.DB, a Application) (int64, error) {
	res, err := db.Exec(`
		INSERT INTO applications
		(company, reported, role_title, location, comp_range, source,
		 date_reviewed, status, fit, notes, tailored_resume, next_action)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.Company, a.Reported, a.RoleTitle, a.Location, a.CompRange, a.Source,
		a.DateReviewed, a.Status, a.Fit, a.Notes, a.TailoredResume, a.NextAction)
	if err != nil {
		return 0, fmt.Errorf("insert application: %w", err)
	}
	return res.LastInsertId()
}

// ListApplications returns every row, oldest first.
func ListApplications(db *sql.DB) ([]Application, error) {
	rows, err := db.Query(`SELECT ` + appColumns + ` FROM applications ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list applications: %w", err)
	}
	defer rows.Close()

	var apps []Application
	for rows.Next() {
		a, err := scanApplication(rows)
		if err != nil {
			return nil, fmt.Errorf("read row: %w", err)
		}
		apps = append(apps, a)
	}
	return apps, rows.Err()
}

// DeleteApplication removes one row. It reports false if no row had that id.
func DeleteApplication(db *sql.DB, id int64) (bool, error) {
	res, err := db.Exec(`DELETE FROM applications WHERE id = ?`, id)
	if err != nil {
		return false, fmt.Errorf("delete application: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// GetApplication loads one row. The bool is false when no row has that id.
func GetApplication(db *sql.DB, id int64) (Application, bool, error) {
	row := db.QueryRow(`SELECT `+appColumns+` FROM applications WHERE id = ?`, id)
	a, err := scanApplication(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Application{}, false, nil
	}
	if err != nil {
		return Application{}, false, fmt.Errorf("get application: %w", err)
	}
	return a, true, nil
}

// UpdateApplication overwrites every field of the row with a.ID.
// It reports false if no row had that id.
func UpdateApplication(db *sql.DB, a Application) (bool, error) {
	res, err := db.Exec(`
		UPDATE applications SET
			company = ?, reported = ?, role_title = ?, location = ?, comp_range = ?,
			source = ?, date_reviewed = ?, status = ?, fit = ?, notes = ?,
			tailored_resume = ?, next_action = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		a.Company, a.Reported, a.RoleTitle, a.Location, a.CompRange,
		a.Source, a.DateReviewed, a.Status, a.Fit, a.Notes,
		a.TailoredResume, a.NextAction, a.ID)
	if err != nil {
		return false, fmt.Errorf("update application: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}
