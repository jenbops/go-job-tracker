package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestFilterApps(t *testing.T) {
	apps := []Application{
		{Company: "Horizon3 AI", RoleTitle: "Engineering Manager"},
		{Company: "Acme", RoleTitle: "Director"},
	}
	got := filterApps(apps, "horizon")
	if len(got) != 1 || got[0].Company != "Horizon3 AI" {
		t.Fatalf("got %+v", got)
	}
	if len(filterApps(apps, "")) != 2 {
		t.Fatal("empty search should keep every row")
	}
}

func TestCreateBlocksDuplicate(t *testing.T) {
	db := newTestDB(t)
	s, err := newServer(db)
	if err != nil {
		t.Fatal(err)
	}
	h := s.routes()

	post := func(form url.Values) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", "/new", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}

	form := url.Values{"company": {"Acme"}, "role": {"Director"}}
	if rec := post(form); rec.Code != http.StatusSeeOther {
		t.Fatalf("first add: status %d, want 303", rec.Code)
	}
	form.Set("company", "Acme Inc.")
	if rec := post(form); rec.Code != http.StatusOK {
		t.Fatalf("duplicate: status %d, want 200 with a warning", rec.Code)
	}
	form.Set("force", "on")
	if rec := post(form); rec.Code != http.StatusSeeOther {
		t.Fatalf("forced add: status %d, want 303", rec.Code)
	}
	apps, err := ListApplications(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 2 {
		t.Fatalf("got %d rows, want 2", len(apps))
	}
}
func TestSortAppsDateTieBreak(t *testing.T) {
	apps := []Application{
		{ID: 246, DateReviewed: "2026-11-20"},
		{ID: 247, DateReviewed: "2026-11-24"},
		{ID: 248, DateReviewed: "2026-11-24"},
	}
	sortApps(apps, "date", true)
	got := []int64{apps[0].ID, apps[1].ID, apps[2].ID}
	want := []int64{248, 247, 246}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order = %v, want %v", got, want)
		}
	}
}
func TestDelete(t *testing.T) {
	db := newTestDB(t)
	s, err := newServer(db)
	if err != nil {
		t.Fatal(err)
	}
	h := s.routes()
	id, err := AddApplication(db, Application{Company: "Acme", RoleTitle: "Director"})
	if err != nil {
		t.Fatal(err)
	}

	do := func(method, path string) int {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
		return rec.Code
	}
	path := fmt.Sprintf("/delete/%d", id)

	if code := do("GET", path); code != http.StatusMethodNotAllowed {
		t.Fatalf("GET delete: status %d, want 405", code)
	}
	if apps, _ := ListApplications(db); len(apps) != 1 {
		t.Fatal("a GET request must not delete anything")
	}
	if code := do("POST", path); code != http.StatusSeeOther {
		t.Fatalf("POST delete: status %d, want 303", code)
	}
	if apps, _ := ListApplications(db); len(apps) != 0 {
		t.Fatalf("row still present: %+v", apps)
	}
	if code := do("POST", "/delete/9999"); code != http.StatusNotFound {
		t.Fatalf("missing id: status %d, want 404", code)
	}
}
func TestEdit(t *testing.T) {
	db := newTestDB(t)
	s, err := newServer(db)
	if err != nil {
		t.Fatal(err)
	}
	h := s.routes()
	id, err := AddApplication(db, Application{Company: "Acme", RoleTitle: "Director", Status: "Applied"})
	if err != nil {
		t.Fatal(err)
	}

	post := func(path string, form url.Values) int {
		req := httptest.NewRequest("POST", path, strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}
	form := url.Values{
		"company": {"Acme"}, "role": {"Director"},
		"status": {"Interview"}, "notes": {"phone screen done"},
	}
	if code := post(fmt.Sprintf("/edit/%d", id), form); code != http.StatusSeeOther {
		t.Fatalf("edit: status %d, want 303", code)
	}
	a, found, err := GetApplication(db, id)
	if err != nil || !found {
		t.Fatalf("get after edit: found=%v err=%v", found, err)
	}
	if a.Status != "Interview" || a.Notes != "phone screen done" {
		t.Fatalf("row not updated: %+v", a)
	}
	if code := post("/edit/9999", form); code != http.StatusNotFound {
		t.Fatalf("missing id: status %d, want 404", code)
	}
}
func postForm(h http.Handler, path string, form url.Values) *httptest.ResponseRecorder {
	req := httptest.NewRequest("POST", path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestStatusChange(t *testing.T) {
	db := newTestDB(t)
	s, err := newServer(db)
	if err != nil {
		t.Fatal(err)
	}
	h := s.routes()
	id, err := AddApplication(db, Application{Company: "Acme", RoleTitle: "Director", Status: "Applied"})
	if err != nil {
		t.Fatal(err)
	}
	path := fmt.Sprintf("/status/%d", id)

	form := url.Values{"status": {"Interview"}, "back": {"/?q=acme&sort=date"}}
	rec := postForm(h, path, form)
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/?q=acme&sort=date" {
		t.Fatalf("got %d to %q", rec.Code, rec.Header().Get("Location"))
	}
	a, found, err := GetApplication(db, id)
	if err != nil || !found {
		t.Fatalf("get: found=%v err=%v", found, err)
	}
	if a.Status != "Interview" || a.Company != "Acme" {
		t.Fatalf("unexpected row: %+v", a)
	}

	form.Set("back", "//evil.example/")
	if loc := postForm(h, path, form).Header().Get("Location"); loc != "/" {
		t.Fatalf("redirect to another site was allowed: %q", loc)
	}

	form.Set("status", " ")
	if code := postForm(h, path, form).Code; code != http.StatusBadRequest {
		t.Fatalf("empty status: got %d, want 400", code)
	}
}
func TestGetStats(t *testing.T) {
	db := newTestDB(t)
	add := func(company, status, source, date string) {
		t.Helper()
		_, err := AddApplication(db, Application{
			Company: company, RoleTitle: "EM", Status: status, Source: source, DateReviewed: date,
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	add("A", "Applied", "LinkedIn", "2026-11-23")  // a Monday
	add("B", "Applied", "", "2026-11-29")          // the Sunday of that week
	add("C", "Rejected", "LinkedIn", "2026-11-30") // the next Monday

	s, err := GetStats(db)
	if err != nil {
		t.Fatal(err)
	}
	if s.Total != 3 {
		t.Errorf("total = %d, want 3", s.Total)
	}
	if len(s.ByStatus) != 2 || s.ByStatus[0].Label != "Applied" ||
		s.ByStatus[0].N != 2 || s.ByStatus[0].Pct != 100 {
		t.Errorf("by status = %+v", s.ByStatus)
	}
	if len(s.BySource) != 2 || s.BySource[0].Label != "LinkedIn" || s.BySource[1].Label != "(none)" {
		t.Errorf("by source = %+v", s.BySource)
	}
	if len(s.ByWeek) != 2 || s.ByWeek[0].Label != "Week of 11/30/2026" || s.ByWeek[1].N != 2 {
		t.Errorf("by week = %+v", s.ByWeek)
	}
}
