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
	PublicKey *PublicKey `json:"public_key,omitempty"`
	CreatedAt *time.Time `json:"created_at"`
}

type Post struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	Title     string     `json:"title"`
	Text      string     `json:"text"`
	PublishAt *time.Time `json:"publish_at"`
	Username  string     `json:"username"`
}

type DB interface {
	AddUser() (string, error)
	LinkUserKey(userID string, key string) error
	PublicKeyForKey(key string) (*PublicKey, error)
	ListKeysForUser(user *User) ([]*PublicKey, error)

	UserForName(name string) (string, error)
	UserForKey(key string) (*User, error)
	User(userID string) (*User, error)
	ValidateName(name string) bool
	SetUserName(userID string, name string) error

	FindPost(postID string) (*Post, error)
	PostsForUser(userID string) ([]*Post, error)
	FindPostWithTitle(title string, userID string) (*Post, error)
	FindAllPosts(page int) ([]*Post, error)
	InsertPost(userID string, title string, text string, publishAt *time.Time) (*Post, error)
	UpdatePost(postID string, text string, publishAt *time.Time) (*Post, error)
	RemovePosts(postIDs []string) error

	Close() error
}
