# go-job-tracker

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