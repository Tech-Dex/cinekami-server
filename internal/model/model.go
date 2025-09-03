package model

import "time"

// Allowed vote categories.
const (
	CategorySoloFriends = "solo_friends"
	CategoryCouple      = "couple"
	CategoryStreaming   = "streaming"
	CategoryArr         = "arr"
)

var AllowedCategories = map[string]struct{}{
	CategorySoloFriends: {},
	CategoryCouple:      {},
	CategoryStreaming:   {},
	CategoryArr:         {},
}

type Movie struct {
	ID            int64            `json:"id"` // TMDb id
	Title         string           `json:"title"`
	ReleaseDate   time.Time        `json:"release_date"`
	Overview      *string          `json:"overview,omitempty"`
	PosterPath    *string          `json:"poster_path,omitempty"`
	BackdropPath  *string          `json:"backdrop_path,omitempty"`
	Popularity    float64          `json:"popularity"`
	Tallies       map[string]int64 `json:"tallies,omitempty"`
	VotedCategory *string          `json:"voted_category,omitempty"`
}

type Tally struct {
	MovieID  int64  `json:"movie_id"`
	Category string `json:"category"`
	Count    int64  `json:"count"`
}

type Snapshot struct {
	Month   string           `json:"month"` // YYYY-MM
	MovieID int64            `json:"movie_id"`
	Tallies map[string]int64 `json:"tallies"`
	Closed  time.Time        `json:"closed_at"`
	// movie metadata joined for convenience
	Title        string    `json:"title,omitempty"`
	ReleaseDate  time.Time `json:"release_date,omitempty"`
	Overview     *string   `json:"overview,omitempty"`
	PosterPath   *string   `json:"poster_path,omitempty"`
	BackdropPath *string   `json:"backdrop_path,omitempty"`
	Popularity   float64   `json:"popularity,omitempty"`
}
