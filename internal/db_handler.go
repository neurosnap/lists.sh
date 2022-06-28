package internal

import (
	"fmt"
	"io"
	"time"

	"github.com/gliderlabs/ssh"
	"github.com/neurosnap/lists.sh/pkg"
	"github.com/picosh/cms/db"
	"github.com/picosh/cms/util"
	sendutils "github.com/picosh/send/utils"
)

type Opener struct {
	entry *sendutils.FileEntry
}

func (o *Opener) Open(name string) (io.Reader, error) {
	return o.entry.Reader, nil
}

type DbHandler struct {
	user   *db.User
	dbpool db.DB
	cfg    *ConfigSite
}

func NewDbHandler(dbpool db.DB) *DbHandler {
	return &DbHandler{
		dbpool: dbpool,
	}
}

func (h *DbHandler) Validate(s ssh.Session) error {
	var err error
	key, err := util.KeyText(s)
	if err != nil {
		return fmt.Errorf("key not found")
	}

	user, err := h.dbpool.FindUserForKey(s.User(), key)
	if err != nil {
		return err
	}

	if user.Name == "" {
		return fmt.Errorf("must have username set")
	}

	h.user = user
	return nil
}

func (h *DbHandler) Write(s ssh.Session, entry *sendutils.FileEntry) error {
	logger := h.cfg.Logger
	userID := h.user.ID
	filename := SanitizeFileExt(entry.Name)
	title := filename

	post, err := h.dbpool.FindPostWithFilename(filename, userID)
	if err != nil {
		logger.Debug("unable to load post, continuing:", err)
	}

	var text string
	if b, err := io.ReadAll(entry.Reader); err == nil {
		text = string(b)
	}

	if !IsTextFile(text, entry.Filepath) {
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

		err := h.dbpool.RemovePosts([]string{post.ID})
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
		_, err = h.dbpool.InsertPost(userID, filename, title, text, description, &publishAt)
		if err != nil {
			return fmt.Errorf("error for %s: %v", filename, err)
		}
	} else {
		publishAt := post.PublishAt
		if parsedText.MetaData.PublishAt != nil {
			publishAt = parsedText.MetaData.PublishAt
		}
		if text == post.Text {
			logger.Infof("(%s) found, but text is identical, skipping", filename)
			return nil
		}

		logger.Infof("(%s) found, updating record", filename)
		_, err = h.dbpool.UpdatePost(post.ID, title, text, description, publishAt)
		if err != nil {
			return fmt.Errorf("error for %s: %v", filename, err)
		}
	}

	return nil
}
