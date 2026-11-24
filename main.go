package main

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"net/http" // add this line
	"os"
	"time"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: go-job-tracker <add|list|check|import|export|sync|serve> [flags]")
	}
	path, err := defaultDBPath()
	if err != nil {
		return err
	}
	db, err := OpenStore(path)
	if err != nil {
		return err
	}
	defer db.Close()

	switch args[0] {
	case "add":
		return runAdd(db, args[1:])
	case "list":
		return runList(db)
	case "check":
		return runCheck(db, args[1:])
	case "import":
		return runImport(db, args[1:])
	case "export":
		return runExport(db, args[1:])
	case "serve":
		return runServe(db, args[1:])
	case "sync":
		return runSync(db, args[1:])
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runAdd(db *sql.DB, args []string) error {
	fs := flag.NewFlagSet("add", flag.ContinueOnError)
	var a Application
	fs.StringVar(&a.Company, "company", "", "company name (required)")
	fs.StringVar(&a.RoleTitle, "role", "", "role title (required)")
	fs.StringVar(&a.Location, "location", "", "job location")
	fs.StringVar(&a.Source, "source", "", "where you found it")
	fs.StringVar(&a.Status, "status", "Applied", "application status")
	fs.StringVar(&a.Reported, "reported", "Yes", "whether this was Reported Yes or No")
	force := fs.Bool("force", false, "add even if the company is already tracked")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if a.Company == "" || a.RoleTitle == "" {
		return errors.New("-company and -role are required")
	}
	if !*force {
		matches, err := FindDuplicates(db, a.Company, a.RoleTitle)
		if err != nil {
			return err
		}
		if len(matches) > 0 {
			printMatches(matches)
			return errors.New("company already tracked; rerun with -force to add anyway")
		}
	}
	a.DateReviewed = time.Now().Format("2006-01-02")

	id, err := AddApplication(db, a)
	if err != nil {
		return err
	}
	fmt.Printf("added #%d: %s, %s\n", id, a.Company, a.RoleTitle)
	return nil
}

func runList(db *sql.DB) error {
	apps, err := ListApplications(db)
	if err != nil {
		return err
	}
	for _, a := range apps {
		fmt.Printf("%-4d %-24s %-36s %s\n", a.ID, a.Company, a.RoleTitle, a.Status)
	}
	fmt.Printf("%d applications\n", len(apps))
	return nil
}
func runCheck(db *sql.DB, args []string) error {
	if len(args) < 1 {
		return errors.New(`usage: check "Company" ["Role"]`)
	}
	role := ""
	if len(args) > 1 {
		role = args[1]
	}
	matches, err := FindDuplicates(db, args[0], role)
	if err != nil {
		return err
	}
	printMatches(matches)
	return nil
}

func printMatches(matches []Match) {
	if len(matches) == 0 {
		fmt.Println("no existing entries for this company")
		return
	}
	for _, m := range matches {
		note := ""
		if m.SameRole {
			note = "  <-- same role"
		}
		fmt.Printf("#%-4d %-24s %-36s %-11s %s%s\n",
			m.App.ID, m.App.Company, m.App.RoleTitle, m.App.DateReviewed, m.App.Status, note)
	}
}
func runImport(db *sql.DB, args []string) error {
	if len(args) != 1 {
		return errors.New("usage: import FILE.csv")
	}
	f, err := os.Open(args[0])
	if err != nil {
		return err
	}
	defer f.Close()
	added, skipped, err := ImportCSV(db, f)
	if err != nil {
		return err
	}
	fmt.Printf("imported %d rows, skipped %d (blank, divider, or already present)\n", added, skipped)
	return nil
}

func runExport(db *sql.DB, args []string) error {
	if len(args) != 1 {
		return errors.New("usage: export FILE.csv")
	}
	f, err := os.Create(args[0])
	if err != nil {
		return err
	}
	defer f.Close()
	return ExportCSV(db, f)
}

func runSync(db *sql.DB, args []string) error {
	fs := flag.NewFlagSet("sync", flag.ContinueOnError)
	dry := fs.Bool("dry-run", false, "show differences without changing the database")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: sync [-dry-run] FILE.csv")
	}
	f, err := os.Open(fs.Arg(0))
	if err != nil {
		return err
	}
	defer f.Close()

	rep, err := SyncCSV(db, f, *dry)
	if err != nil {
		return err
	}
	verb := "added"
	if *dry {
		verb = "would add"
	}
	for _, a := range rep.New {
		fmt.Printf("%s: %s, %s (%s)\n", verb, a.Company, a.RoleTitle, a.Status)
	}
	for _, c := range rep.Changed {
		fmt.Println("changed:", c)
	}
	for _, a := range rep.OnlyDB {
		fmt.Printf("only in database (kept): #%d %s, %s\n", a.ID, a.Company, a.RoleTitle)
	}
	fmt.Printf("%d new, %d changed, %d unchanged, %d only in database\n",
		len(rep.New), len(rep.Changed), rep.Same, len(rep.OnlyDB))
	return nil
}

func runServe(db *sql.DB, args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	addr := fs.String("addr", "127.0.0.1:8080", "address to listen on")
	if err := fs.Parse(args); err != nil {
		return err
	}
	s, err := newServer(db)
	if err != nil {
		return err
	}
	fmt.Printf("listening on http://%s (Ctrl+C to stop)\n", *addr)
	return http.ListenAndServe(*addr, s.routes())
}
