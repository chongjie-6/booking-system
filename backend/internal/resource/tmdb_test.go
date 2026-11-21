package resource

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestFetchTMDB checks the token is sent, a null poster becomes "", the official YouTube trailer is picked
// over teasers, other sites and unofficial trailers, and a non-200 answer is an error.
func TestFetchTMDB(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" || r.URL.Path != "/movie/862" ||
			r.URL.Query().Get("append_to_response") != "videos" {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte(`{"overview":"Toys.","runtime":81,"poster_path":null,"backdrop_path":"/b.jpg","videos":{"results":[
			{"key":"teaser","site":"YouTube","type":"Teaser","official":true},
			{"key":"vimeo","site":"Vimeo","type":"Trailer","official":true},
			{"key":"fan","site":"YouTube","type":"Trailer","official":false},
			{"key":"official","site":"YouTube","type":"Trailer","official":true}
		]}}`))
	}))
	defer srv.Close()
	tmdbAPI = srv.URL

	m, err := fetchTMDB(context.Background(), "tok", 862)
	if err != nil || m != (tmdbMovie{Overview: "Toys.", Runtime: 81, BackdropPath: "/b.jpg", TrailerKey: "official"}) {
		t.Fatalf("fetchTMDB = %+v, %v", m, err)
	}
	if _, err := fetchTMDB(context.Background(), "tok", 1); err == nil {
		t.Fatal("fetchTMDB of a missing film = nil error")
	}
}
