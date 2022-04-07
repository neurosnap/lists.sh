package scp

import (
	"database/sql"
	"io"
	"log"
	"time"

	"github.com/gliderlabs/ssh"
)

type DbHandler struct{}

func (h *DbHandler) Write(_ ssh.Session, entry *FileEntry, dbpool *sql.DB) error {
	personaId := "8c4de632-e27a-491f-8c07-877349c91600"
	var id string
	err := dbpool.QueryRow(
		"SELECT id FROM posts WHERE title = $1 AND persona_id = $2",
		entry.Filepath,
		personaId,
	).Scan(&id)
	if err != nil {
		log.Println(err)
	}

	var text string
	if b, err := io.ReadAll(entry.Reader); err == nil {
		text = string(b)
	}

	log.Println(id)
	if id == "" {
		log.Printf("%s not found, adding record", entry.Filepath)
		_, err := dbpool.Exec(
			"INSERT INTO posts (persona_id, title, text) VALUES ($1, $2, $3)",
			personaId,
			entry.Filepath,
			text,
		)
		if err != nil {
			log.Printf("error for %s: %v", entry.Filepath, err)
		}
	} else {
		log.Printf("%s found, updating record", entry.Filepath)
		_, err := dbpool.Exec(
			"UPDATE posts SET text = $1, updated_at = $2 WHERE id = $3",
			text,
			time.Now(),
			id,
		)
		if err != nil {
			log.Printf("error for %s: %v", entry.Filepath, err)
		}
	}

	return nil
}
