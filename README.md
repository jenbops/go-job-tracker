# go-job-tracker

A small job application tracker written in Go. It keeps applications in a
local SQLite file and has a command line tool and a local web page.

## Requirements

Go. The version is in `go.mod`. The SQLite driver is pure Go, so no C
compiler is needed.

## Command line

Run these from the project folder:

    go run . add -company "Acme" -role "Engineering Manager" -source LinkedIn
    go run . list
    go run . check "Acme" "Engineering Manager"
    go run . import sheet.csv
    go run . export backup.csv
    go run . sync -dry-run sheet.csv
    go run . sync sheet.csv
    go run . serve

### add

Flags: `-company` and `-role` (both required), `-location`, `-source`,
`-status` (default Applied), `-reported` (Yes or No, default Yes) and
`-force`. The date reviewed is set to today. Pay range, fit, notes, tailored
materials and next action have no flag; add them afterwards with the edit
form on the web page.

`add` stops when the company is already tracked. "Horizon3 AI",
"Horizon3.ai" and "Horizon3 AI, Inc." all count as the same company. Add
`-force` to add the row anyway.

### check

Lists every existing row for a company and marks the ones with the same
role. The role is optional.

### list

Prints the id, company, role and status of every row, then a total.

### import and export

`import` reads a CSV file and adds the rows that are not already there. A
row is skipped when the company or role is empty, or when a row with the same
company, role and date reviewed exists, so running it twice is safe. The
Yes/No column is read from a header named `reported`. If your sheet uses a
different header, add it to `columnAliases` from a file ending in
`_local.go`. Those files are git-ignored, so the name stays on your machine.

`export` writes every row to a CSV file in the same column order as the
sheet, with the column named `reported`.

### sync

`sync` compares a fresh sheet export with the database. Rows match on
company, role and date reviewed. New rows are added and changed rows are
updated to match the sheet (the sheet wins). Rows only in the database are
listed but never deleted. Run it with `-dry-run` first to see the changes
without making them.

## Web page

    go run . serve

Then open http://127.0.0.1:8080. The page has:

- a searchable table you can sort by clicking the column headings
- a form to add an application, with a live warning when the company is
  already tracked
- an edit form, and a delete button that asks first
- a status menu on each row, so you can change a status without opening the
  edit form
- a stats page at `/stats` with counts by status, source and week

The server listens on 127.0.0.1 only, so nothing else on your network can
reach it. Use `-addr` to change that, and only if you understand the risk,
since the data is personal.

## Where the data lives

On macOS the database is at
`~/Library/Application Support/go-job-tracker/jobtracker.db`. Other systems
use their standard config folder. Set `JOBTRACKER_DB` to use a different
file, for example a practice copy:

    JOBTRACKER_DB=/tmp/practice.db go run . serve

The database and CSV files are git-ignored so personal data stays out of
the repo.

## Tests

    go test ./...
