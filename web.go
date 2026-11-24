package main

import (
	"database/sql"
	"embed"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"
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
	mux.HandleFunc("GET /new", s.handleNewForm)
	mux.HandleFunc("POST /new", s.handleCreate)
	mux.HandleFunc("GET /api/check", s.handleCheck)
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
		less = func(i, j int) bool {
			if apps[i].DateReviewed != apps[j].DateReviewed {
				return apps[i].DateReviewed < apps[j].DateReviewed
			}
			return apps[i].ID < apps[j].ID
		}
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

var statusOptions = []string{
	"Applied", "Interview", "Screening Interview Scheduled",
	"Screening Interview Complete", "Hiring Manager Interview",
	"Sent Thank you for screening interview", "Passed", "Rejected",
	"Closed/Removed", "Eliminated",
}

type formPage struct {
	App      Application
	Error    string
	Matches  []Match
	Statuses []string
}

func (s *server) renderForm(w http.ResponseWriter, p formPage) {
	p.Statuses = statusOptions
	if err := s.tmpl.ExecuteTemplate(w, "form.html", p); err != nil {
		log.Println("render:", err)
	}
}

func (s *server) handleNewForm(w http.ResponseWriter, r *http.Request) {
	s.renderForm(w, formPage{App: Application{
		Status:       "Applied",
		Reported:     "Yes",
		DateReviewed: time.Now().Format("2006-01-02"),
	}})
}

func (s *server) handleCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	v := func(name string) string { return strings.TrimSpace(r.FormValue(name)) }
	a := Application{
		Company:        v("company"),
		RoleTitle:      v("role"),
		Location:       v("location"),
		CompRange:      v("comp"),
		Source:         v("source"),
		DateReviewed:   normalizeDate(v("date")),
		Status:         v("status"),
		Reported:       v("reported"),
		Fit:            v("fit"),
		Notes:          v("notes"),
		TailoredResume: v("tailored"),
		NextAction:     v("next"),
	}
	if a.Company == "" || a.RoleTitle == "" {
		s.renderForm(w, formPage{App: a, Error: "Company and role are required."})
		return
	}
	if a.DateReviewed == "" {
		a.DateReviewed = time.Now().Format("2006-01-02")
	}
	if r.FormValue("force") == "" {
		matches, err := FindDuplicates(s.db, a.Company, a.RoleTitle)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if len(matches) > 0 {
			s.renderForm(w, formPage{
				App:     a,
				Matches: matches,
				Error:   "This company is already tracked. Check the box below to add it anyway.",
			})
			return
		}
	}
	if _, err := AddApplication(s.db, a); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *server) handleCheck(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	matches, err := FindDuplicates(s.db, q.Get("company"), q.Get("role"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	type item struct {
		ID       int64  `json:"id"`
		Company  string `json:"company"`
		Role     string `json:"role"`
		Date     string `json:"date"`
		Status   string `json:"status"`
		SameRole bool   `json:"sameRole"`
	}
	out := make([]item, 0, len(matches))
	for _, m := range matches {
		out = append(out, item{m.App.ID, m.App.Company, m.App.RoleTitle,
			displayDate(m.App.DateReviewed), m.App.Status, m.SameRole})
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(out); err != nil {
		log.Println("encode:", err)
	}
}
