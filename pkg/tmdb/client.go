package tmdb

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type Client struct {
	APIKey  string
	BaseURL string
	Client  *http.Client
}

type Movie struct {
	TMDBID       int32
	Title        string
	ReleaseDate  time.Time
	Overview     string
	PosterPath   string
	BackdropPath string
	Popularity   float64
}

type discoverResp struct {
	Page       int            `json:"page"`
	TotalPages int            `json:"total_pages"`
	Results    []discoverItem `json:"results"`
}

type discoverItem struct {
	ID           int32   `json:"id"`
	Title        string  `json:"title"`
	ReleaseDate  string  `json:"release_date"`
	Overview     string  `json:"overview"`
	PosterPath   string  `json:"poster_path"`
	BackdropPath string  `json:"backdrop_path"`
	Popularity   float64 `json:"popularity"`
}

func New(apiKey string) *Client {
	return &Client{APIKey: apiKey, BaseURL: "https://api.themoviedb.org/3", Client: &http.Client{Timeout: 15 * time.Second}}
}

// DiscoverByReleaseWindow fetches movies with a primary_release_date between start and end (inclusive).
// If maxPages <= 0, fetch all pages; otherwise stop at maxPages.
func (c *Client) DiscoverByReleaseWindow(start, end time.Time, region, language string, maxPages int) ([]Movie, error) {
	if c.APIKey == "" {
		return nil, fmt.Errorf("missing TMDB API key")
	}
	var out []Movie
	page := 1
	done := false
	for {
		u, _ := url.Parse(c.BaseURL + "/discover/movie")
		q := u.Query()
		q.Set("api_key", c.APIKey)
		if region != "" {
			q.Set("region", region)
		}
		q.Set("with_release_type", "3|2|1") // Premiere, Theatrical Limited, Theatrical
		q.Set("primary_release_date.gte", start.Format("2006-01-02"))
		q.Set("primary_release_date.lte", end.Format("2006-01-02"))
		if language != "" {
			q.Set("language", language)
		}
		q.Set("sort_by", "popularity.desc")
		q.Set("page", strconv.Itoa(page))
		u.RawQuery = q.Encode()

		req, _ := http.NewRequest(http.MethodGet, u.String(), nil)
		resp, err := c.Client.Do(req)
		if err != nil {
			return nil, err
		}
		func() {
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				err = fmt.Errorf("tmdb status %d", resp.StatusCode)
				return
			}
			var dr discoverResp
			if e := json.NewDecoder(resp.Body).Decode(&dr); e != nil {
				err = e
				return
			}
			for _, it := range dr.Results {
				if it.ReleaseDate == "" {
					continue
				}
				d, e := time.Parse("2006-01-02", it.ReleaseDate)
				if e != nil {
					continue
				}
				out = append(out, Movie{TMDBID: it.ID, Title: it.Title, ReleaseDate: d, Overview: it.Overview, PosterPath: it.PosterPath, BackdropPath: it.BackdropPath, Popularity: it.Popularity})
			}
			// Determine if we're done fetching pages
			if (maxPages > 0 && page >= maxPages) || dr.Page >= dr.TotalPages {
				done = true
				return
			}
			page++
		}()
		if err != nil {
			return nil, err
		}
		if done {
			break
		}
	}
	return out, nil
}
