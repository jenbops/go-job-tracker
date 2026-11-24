package main

import (
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
