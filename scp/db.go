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
	title := internal.SanitizeFileExt(entry.Name)
	post, err := dbpool.FindPostWithTitle(title, userID)

	var text string
	if b, err := io.ReadAll(entry.Reader); err == nil {
		text = string(b)
	}

	if !internal.IsTextFile(text, entry.Filepath) {
		return fmt.Errorf("file must be a text file")
	}

	if post == nil {
		log.Printf("%s not found, adding record", title)
		post, err = dbpool.InsertPost(userID, title, text)
		if err != nil {
			return fmt.Errorf("error for %s: %v", title, err)
		}
	} else {
		log.Printf("%s found, updating record", title)
		post, err = dbpool.UpdatePost(post.ID, text)
		if err != nil {
			return fmt.Errorf("error for %s: %v", title, err)
		}
	}

	return nil
}
