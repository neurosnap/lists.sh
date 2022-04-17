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
	PublicKey *PublicKey `json:"public_key,omitempty"`
	Personas  []*Persona  `json:"personas"`
	CreatedAt *time.Time `json:"created_at"`
}

type Persona struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	CreatedAt *time.Time `json:"created_at"`
}

type Post struct {
	ID        string     `json:"id"`
	PersonaID string     `json:"persona_id"`
	Title     string     `json:"title"`
	Text      string     `json:"text"`
	PublishAt *time.Time `json:"publish_at"`
}

type DB interface {
	AddUser() (string, error)
	LinkUserKey(user *User, key string) error
	PublicKeyForKey(key string) (*PublicKey, error)
	ListKeysForUser(user *User) ([]*PublicKey, error)

	UserForKey(key string) (*User, error)
	User(userID string) (*User, error)
	ValidateName(name string) bool

	ListPersonas(userID string) ([]*Persona, error)
	AddPersona(userID string, persona string) (*Persona, error)
	RemovePersona(persona string) error

	FindPost(postID string) (*Post, error)
    PostsForUser(userID string) ([]*Post, error)
	FindPostWithTitle(title string, personaID string) (*Post, error)
	InsertPost(personaID string, title string, text string) (*Post, error)
	UpdatePost(postID string, text string) (*Post, error)
	RemovePosts(postIDs []string) error

	Close() error
}
