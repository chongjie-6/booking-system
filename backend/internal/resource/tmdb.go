package resource

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

var tmdbAPI = "https://api.themoviedb.org/3"

var tmdbClient = &http.Client{Timeout: 10 * time.Second}

// tmdbMovie is the part of TMDB's GET /movie/{id} we keep. PosterPath is "" when TMDB has no poster.
type tmdbMovie struct {
	Overview   string `json:"overview"`
	Runtime    int    `json:"runtime"`
	PosterPath string `json:"poster_path"`
}

// FillFromTMDB fills in the synopsis, runtime and poster of movies with a TMDB ID that haven't been fetched yet,
// at startup and then every minute until ctx is cancelled, so freshly seeded films get posters without a restart.
// Does nothing without TMDB_TOKEN (TMDB's API read access token). Takes a context; returns once it's cancelled.
func FillFromTMDB(ctx context.Context) {
	token := os.Getenv("TMDB_TOKEN")
	if token == "" {
		log.Print("TMDB_TOKEN not set; films keep their seeded details and drawn posters")
		return
	}
	tick := time.NewTicker(time.Minute)
	defer tick.Stop()
	for {
		ids, err := dbUnfetchedMovies(ctx)
		if err != nil {
			log.Printf("tmdb: %v", err)
		}
		for movieID, tmdbID := range ids {
			m, err := fetchTMDB(ctx, token, tmdbID)
			if err == nil {
				err = dbSetTMDB(ctx, movieID, m)
			}
			if err != nil {
				log.Printf("tmdb movie %d: %v", tmdbID, err)
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
	}
}

// fetchTMDB gets one film's details from TMDB. Takes a context, the token and the TMDB ID; returns the details or the error.
func fetchTMDB(ctx context.Context, token string, tmdbID int) (tmdbMovie, error) {
	var m tmdbMovie
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/movie/%d", tmdbAPI, tmdbID), nil)
	if err != nil {
		return m, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := tmdbClient.Do(req)
	if err != nil {
		return m, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return m, fmt.Errorf("TMDB answered %s", res.Status)
	}
	return m, json.NewDecoder(res.Body).Decode(&m)
}
