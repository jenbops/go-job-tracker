package main

import (
	"database/sql"
	"embed"
	"html/template"
	"log"
	"net/http"
	"sort"
	"strings"
)

//go:embed templates/*.html
var templateFS embed.FS

type server struct {
	db   *sql.DB
	tmpl *template.Template
}

func newServer(db *sql.DB) (*server, error) {
	funcs := template.FuncMap{"date": displayDate, "chip": chipClass}
	t, err := template.New("").Funcs(funcs).ParseFS(templateFS, "templates/*.html")
	if err != nil {
		return nil, err
	}
	return &server{db: db, tmpl: t}, nil
}

func (s *server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.handleIndex)
	return mux
}

func (s *server) handleIndex(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	sortBy := r.URL.Query().Get("sort")
	desc := r.URL.Query().Get("dir") == "desc"
	if sortBy == "" {
		sortBy, desc = "date", true // newest first by default
	}

	all, err := ListApplications(s.db)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	apps := filterApps(all, q)
	sortApps(apps, sortBy, desc)

	nextDir := "desc"
	if desc {
		nextDir = "asc"
	}
	data := struct {
		Apps    []Application
		Query   string
		NextDir string
		Total   int
	}{apps, q, nextDir, len(all)}

	if err := s.tmpl.ExecuteTemplate(w, "index.html", data); err != nil {
		log.Println("render:", err)
	}
}

// filterApps keeps rows where any main text field contains q, ignoring case.
func filterApps(apps []Application, q string) []Application {
	q = strings.ToLower(q)
	if q == "" {
		return apps
	}
	var out []Application
	for _, a := range apps {
		fields := []string{a.Company, a.RoleTitle, a.Location, a.Source, a.Status, a.Notes, a.NextAction}
		if strings.Contains(strings.ToLower(strings.Join(fields, " ")), q) {
			out = append(out, a)
		}
	}
	return out
}

// sortApps sorts in place by a known column name. Unknown names sort by id.
func sortApps(apps []Application, by string, desc bool) {
	var less func(i, j int) bool
	switch by {
	case "company":
		less = func(i, j int) bool { return strings.ToLower(apps[i].Company) < strings.ToLower(apps[j].Company) }
	case "role":
		less = func(i, j int) bool { return strings.ToLower(apps[i].RoleTitle) < strings.ToLower(apps[j].RoleTitle) }
	case "date":
		less = func(i, j int) bool { return apps[i].DateReviewed < apps[j].DateReviewed }
	case "status":
		less = func(i, j int) bool { return strings.ToLower(apps[i].Status) < strings.ToLower(apps[j].Status) }
	default:
		less = func(i, j int) bool { return apps[i].ID < apps[j].ID }
	}
	sort.SliceStable(apps, func(i, j int) bool {
		if desc {
			return less(j, i)
		}
		return less(i, j)
	})
}

// chipClass picks a Cascading Style Sheets (CSS) class for a status badge.
func chipClass(status string) string {
	s := strings.ToLower(status)
	switch {
	case strings.Contains(s, "interview"):
		return "chip-interview"
	case s == "applied":
		return "chip-applied"
	case s == "rejected", s == "passed", s == "eliminated", strings.HasPrefix(s, "closed"):
		return "chip-closed"
	}
	return "chip-other"
}
