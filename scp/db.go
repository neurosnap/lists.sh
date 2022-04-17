package scp

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"

	"github.com/gliderlabs/ssh"
	"github.com/neurosnap/lists.sh/internal/db"
)

func keyText(s ssh.Session) (string, error) {
	if s.PublicKey() == nil {
		return "", fmt.Errorf("Session doesn't have public key")
	}
	kb := base64.StdEncoding.EncodeToString(s.PublicKey().Marshal())
	return fmt.Sprintf("%s %s", s.PublicKey().Type(), kb), nil
}

type DbHandler struct{}

func (h *DbHandler) Delete(s ssh.Session, files []string, dbpool db.DB) error {
	key, err := keyText(s)
	if err != nil {
		return err
	}

	user, err := dbpool.UserForKey(key)
	if err != nil {
		return err
	}

	posts, err := dbpool.PostsForUser(user.ID)
	toDelete := []string{}
	for _, post := range posts {
		found := false
		for _, file := range files {
			if post.Title == file {
				found = true
			}
		}
		if !found {
			toDelete = append(toDelete, post.ID)
		}
	}

	if len(toDelete) > 0 {
		err = dbpool.RemovePosts(toDelete)
		if err != nil {
			log.Println(err)
		}
	}
	return err
}

func (h *DbHandler) Write(s ssh.Session, entry *FileEntry, dbpool db.DB) error {
	key, err := keyText(s)
	if err != nil {
		return err
	}

	user, err := dbpool.UserForKey(key)
	if err != nil {
		return err
	}

	if len(user.Personas) == 0 {
		return fmt.Errorf("User must set a username before publishing content")
	}

	personaID := user.Personas[0].ID

	post, err := dbpool.FindPostWithTitle(entry.Filepath, personaID)
	if err != nil {
		log.Println(err)
	}

	var text string
	if b, err := io.ReadAll(entry.Reader); err == nil {
		text = string(b)
	}

	if post == nil {
		log.Printf("%s not found, adding record", entry.Filepath)
		post, err = dbpool.InsertPost(personaID, entry.Filepath, text)
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
