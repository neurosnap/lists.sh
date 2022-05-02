package scp

import (
	"fmt"
	"io"
	"time"

	"github.com/gliderlabs/ssh"
	"github.com/neurosnap/lists.sh/internal"
	"github.com/neurosnap/lists.sh/internal/db"
	"github.com/neurosnap/lists.sh/pkg"
)

type Opener struct {
	entry *FileEntry
}

func (o *Opener) Open(name string) (io.Reader, error) {
	return o.entry.Reader, nil
}

type DbHandler struct{}

func (h *DbHandler) Write(s ssh.Session, entry *FileEntry, user *db.User, dbpool db.DB) error {
	logger := internal.CreateLogger()
	userID := user.ID
	filename := internal.SanitizeFileExt(entry.Name)
	title := filename
	post, err := dbpool.FindPostWithFilename(filename, userID)

	var text string
	if b, err := io.ReadAll(entry.Reader); err == nil {
		text = string(b)
	}

	if !internal.IsTextFile(text, entry.Filepath) {
		return fmt.Errorf("WARNING: (%s) invalid file, format must be '.txt' and the contents must be plain text, skipping", entry.Name)
	}

	parsedText := pkg.ParseText(text)
	if parsedText.MetaData.Title != "" {
		title = parsedText.MetaData.Title
	}
	description := parsedText.MetaData.Description

	// if the file is empty we remove it from our database
	if len(text) == 0 {
		// skip empty files from being added to db
		if post == nil {
			logger.Infof("(%s) is empty, skipping record", filename)
			return nil
		}

		err := dbpool.RemovePosts([]string{post.ID})
		logger.Infof("(%s) is empty, removing record", filename)
		if err != nil {
			return fmt.Errorf("error for %s: %v", filename, err)
		}
	} else if post == nil {
		publishAt := time.Now()
		if parsedText.MetaData.PublishAt != nil {
			publishAt = *parsedText.MetaData.PublishAt
		}
		logger.Infof("(%s) not found, adding record", filename)
		post, err = dbpool.InsertPost(userID, filename, title, text, description, &publishAt)
		if err != nil {
			return fmt.Errorf("error for %s: %v", filename, err)
		}
	} else {
		publishAt := post.PublishAt
		if parsedText.MetaData.PublishAt != nil {
			publishAt = parsedText.MetaData.PublishAt
		}
		logger.Infof("(%s) found, updating record", filename)
		post, err = dbpool.UpdatePost(post.ID, title, text, description, publishAt)
		if err != nil {
			return fmt.Errorf("error for %s: %v", filename, err)
		}
	}

	return nil
}
