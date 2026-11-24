# go-job-tracker

A small job application tracker written in Go. It keeps applications in a
local SQLite file and has a command line tool and a local web page.

## Requirements

Go. The version is in `go.mod`.

## Command line

    go run . add -company "Acme" -role "Engineering Manager" -source LinkedIn
    go run . list
    go run . check "Acme" "Engineering Manager"
    go run . import sheet.csv
    go run . export backup.csv

`add` stops when the company is already tracked. "Horizon3 AI",
"Horizon3.ai" and "Horizon3 AI, Inc." all count as the same company.
Add `-force` to add the row anyway.

## Web page

    go run . serve

Then open http://127.0.0.1:8080. The page has a searchable, sortable table,
a form to add an application with a live warning when the company is already
tracked, an edit form, and a delete button. A stats page at /stats/ shows counts
by status, source and week.

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
A small job application tracker written in Go. It keeps applications in a
local SQLite file and runs from the command line.

## Requirements

Go. The version is in `go.mod`.

## Usage

    go run . add -company "Acme" -role "Engineering Manager" -source LinkedIn
    go run . list
    go run . check "Acme" "Engineering Manager"
    go run . import sheet.csv
    go run . export backup.csv

`add` stops when the company is already tracked. "Horizon3 AI",
"Horizon3.ai" and "Horizon3 AI, Inc." all count as the same company.
Add `-force` to add the row anyway.

## Where the data lives

On macOS the database is at
`~/Library/Application Support/go-job-tracker/jobtracker.db`. Other systems
use their standard config folder. Set `JOBTRACKER_DB` to use a different
file. The database and CSV files are git-ignored so personal data stays out
of the repo.

## Tests

    go test ./...
