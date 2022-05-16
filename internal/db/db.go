package db

import (
	"errors"
	"regexp"
	"time"
)

var ErrNameTaken = errors.New("name taken")

type PublicKey struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	Key       string     `json:"key"`
	CreatedAt *time.Time `json:"created_at"`
}

type User struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	PublicKey *PublicKey `json:"public_key,omitempty"`
	CreatedAt *time.Time `json:"created_at"`
}

type Post struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	Filename    string     `json:"filename"`
	Title       string     `json:"title"`
	Text        string     `json:"text"`
	Description string     `json:"description"`
	PublishAt   *time.Time `json:"publish_at"`
	Username    string     `json:"username"`
}

type Paginate[T any] struct {
	Data  []T
	Total int
}

type Analytics struct {
	TotalUsers     int
	UsersLastMonth int
	TotalPosts     int
	PostsLastMonth int
	UsersWithPost  int
}

type Pager struct {
	Num  int
	Page int
}

type ErrMultiplePublicKeys struct{}

func (m *ErrMultiplePublicKeys) Error() string {
	return "there are multiple users with this public key, you must provide the username when using SSH: `ssh <user>@lists.sh`\n"
}

var NameValidator = regexp.MustCompile("^[a-zA-Z0-9]{1,50}$")
var DenyList = []string{"admin", "abuse", "cgi", "ops", "help", "spec", "root"}

type DB interface {
	AddUser() (string, error)
	RemoveUsers(userIDs []string) error
	LinkUserKey(userID string, key string) error
	PublicKeyForKey(key string) (*PublicKey, error)
	ListKeysForUser(user *User) ([]*PublicKey, error)
	RemoveKeys(keyIDs []string) error

	SiteAnalytics() (*Analytics, error)

	UserForName(name string) (*User, error)
	UserForNameAndKey(name string, key string) (*User, error)
	UserForKey(key string) (*User, error)
	User(userID string) (*User, error)
	ValidateName(name string) bool
	SetUserName(userID string, name string) error

	FindPost(postID string) (*Post, error)
	PostsForUser(userID string) ([]*Post, error)
	FindPostWithFilename(filename string, userID string) (*Post, error)
	FindAllPosts(pager *Pager) (*Paginate[*Post], error)
	InsertPost(userID string, filename string, title string, text string, description string, publishAt *time.Time) (*Post, error)
	UpdatePost(postID string, title string, text string, description string, publishAt *time.Time) (*Post, error)
	RemovePosts(postIDs []string) error

	Close() error
}
