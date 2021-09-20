package server

import "time"

type Message struct {
	ID         string    `json:"id"`
	Content    string    `json:"content"`
	AuthorName string    `json:"author_name"`
	AuthorID   string    `json:"author_id,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	Edited     bool      `json:"edited"`
}
