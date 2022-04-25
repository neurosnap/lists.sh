package db

import (
	"errors"
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
	Bio       string     `json:"bio"`
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
}

type DB interface {
	AddUser() (string, error)
	LinkUserKey(userID string, key string) error
	PublicKeyForKey(key string) (*PublicKey, error)
	ListKeysForUser(user *User) ([]*PublicKey, error)

	SiteAnalytics() (*Analytics, error)

	UserForName(name string) (*User, error)
	UserForKey(key string) (*User, error)
	User(userID string) (*User, error)
	ValidateName(name string) bool
	SetUserName(userID string, name string) error

	FindPost(postID string) (*Post, error)
	PostsForUser(userID string) ([]*Post, error)
	FindPostWithFilename(filename string, userID string) (*Post, error)
	FindAllPosts(page int) (*Paginate[*Post], error)
	InsertPost(userID string, filename string, title string, text string, description string, publishAt *time.Time) (*Post, error)
	UpdatePost(postID string, title string, text string, description string, publishAt *time.Time) (*Post, error)
	RemovePosts(postIDs []string) error

	Close() error
}
