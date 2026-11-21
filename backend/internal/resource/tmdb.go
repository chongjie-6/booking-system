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

// tmdbMovie is the part of TMDB's GET /movie/{id} we keep. The paths and TrailerKey are "" when TMDB has none.
type tmdbMovie struct {
	Overview     string `json:"overview"`
	Runtime      int    `json:"runtime"`
	PosterPath   string `json:"poster_path"`
	BackdropPath string `json:"backdrop_path"`
	TrailerKey   string `json:"-"`
}

type tmdbVideo struct {
	Key      string `json:"key"`
	Site     string `json:"site"`
	Type     string `json:"type"`
	Official bool   `json:"official"`
}

// FillFromTMDB fills in the synopsis, runtime, poster, backdrop and trailer of movies with a TMDB ID that haven't been fetched yet,
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
	var body struct {
		tmdbMovie
		Videos struct {
			Results []tmdbVideo `json:"results"`
		} `json:"videos"`
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/movie/%d?append_to_response=videos", tmdbAPI, tmdbID), nil)
	if err != nil {
		return body.tmdbMovie, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := tmdbClient.Do(req)
	if err != nil {
		return body.tmdbMovie, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return body.tmdbMovie, fmt.Errorf("TMDB answered %s", res.Status)
	}
	err = json.NewDecoder(res.Body).Decode(&body)
	body.TrailerKey = trailerKey(body.Videos.Results)
	return body.tmdbMovie, err
}

// trailerKey picks the YouTube key of the film's official trailer, else of any trailer; "" if there is none.
func trailerKey(videos []tmdbVideo) string {
	key := ""
	for _, v := range videos {
		if v.Site != "YouTube" || v.Type != "Trailer" {
			continue
		}
		if v.Official {
			return v.Key
		}
		if key == "" {
			key = v.Key
		}
	}
	return key
}
