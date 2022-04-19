package scp

import (
	"fmt"
	"io"
	"log"

	"github.com/gliderlabs/ssh"
	"github.com/neurosnap/lists.sh/internal"
	"github.com/neurosnap/lists.sh/internal/db"
)

type Opener struct {
	entry *FileEntry
}

func (o *Opener) Open(name string) (io.Reader, error) {
	return o.entry.Reader, nil
}

type DbHandler struct{}

func (h *DbHandler) Write(s ssh.Session, entry *FileEntry, user *db.User, dbpool db.DB) error {
	userID := user.ID

	post, err := dbpool.FindPostWithTitle(entry.Name, userID)
	if err != nil {
		log.Println(err)
	}

	var text string
	if b, err := io.ReadAll(entry.Reader); err == nil {
		text = string(b)
	}

	if !internal.IsTextFile(text, entry.Filepath) {
		return fmt.Errorf("File must be a text file")
	}

	if post == nil {
		log.Printf("%s not found, adding record", entry.Filepath)
		post, err = dbpool.InsertPost(userID, entry.Filepath, text)
		if err != nil {
			return fmt.Errorf("error for %s: %v", entry.Filepath, err)
		}
	} else {
		log.Printf("%s found, updating record", entry.Filepath)
		post, err = dbpool.UpdatePost(post.ID, text)
		if err != nil {
			return fmt.Errorf("error for %s: %v", entry.Filepath, err)
		}
	}

	return nil
}
