package model

import "time"

type URL struct {
	ID        int
	Original  string
	ShortCode string
	CreatedAt time.Time
}

type ResultData struct {
	Success  bool
	ShortURL string
	Error    string
}
