package resource

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestFetchTMDB checks the token is sent, a null poster becomes "", and a non-200 answer is an error.
func TestFetchTMDB(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" || r.URL.Path != "/movie/862" {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte(`{"overview":"Toys.","runtime":81,"poster_path":null}`))
	}))
	defer srv.Close()
	tmdbAPI = srv.URL

	m, err := fetchTMDB(context.Background(), "tok", 862)
	if err != nil || m != (tmdbMovie{Overview: "Toys.", Runtime: 81}) {
		t.Fatalf("fetchTMDB = %+v, %v", m, err)
	}
	if _, err := fetchTMDB(context.Background(), "tok", 1); err == nil {
		t.Fatal("fetchTMDB of a missing film = nil error")
	}
}
