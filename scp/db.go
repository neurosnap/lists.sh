package scp

import (
	"io"
	"log"

	"github.com/gliderlabs/ssh"
	"github.com/neurosnap/lists.sh/internal/db"
)

type DbHandler struct{}

func (h *DbHandler) Write(_ ssh.Session, entry *FileEntry, dbpool db.DB) error {
	personaId := "8c4de632-e27a-491f-8c07-877349c91600"
	post, err := dbpool.FindPostWithTitle(entry.Filepath, personaId)
	if err != nil {
		log.Println(err)
	}

	var text string
	if b, err := io.ReadAll(entry.Reader); err == nil {
		text = string(b)
	}

	if post == nil {
		log.Printf("%s not found, adding record", entry.Filepath)
		post, err = dbpool.InsertPost(personaId, entry.Filepath, text)
		if err != nil {
			log.Printf("error for %s: %v", entry.Filepath, err)
		}
	} else {
		log.Printf("%s found, updating record", entry.Filepath)
		post, err = dbpool.UpdatePost(post.ID, text)
		if err != nil {
			log.Printf("error for %s: %v", entry.Filepath, err)
		}
	}

	return nil
}
