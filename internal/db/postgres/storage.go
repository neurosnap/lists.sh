package postgres

import (
	"database/sql"
	"log"

	"github.com/neurosnap/lists.sh/internal/db"
)

const (
	sqlSelectPublicKey     = `SELECT id, user_id, key, created_at FROM public_key WHERE key = $1`
	sqlSelectPublicKeys    = `SELECT id, user_id, key, created_at FROM public_keys WHERE user_id = $1`
	sqlSelectUser          = `SELECT id, created_at FROM app_users WHERE id = $1`
	sqlSelectPersonas      = `SELECT name FROM personas WHERE user_id = $1`
	sqlSelectPostWithTitle = `SELECT id, persona_id, title, text, publish_at FROM posts WHERE title = $1 AND persona_id = $2`
	sqlSelectPost          = `SELECT id, persona_id, title, text, publish_at FROM posts WHERE id = $1`

	sqlInsertPublicKey = `INSERT INTO public_keys (user_id, key) VALUES ($1, $2)`
	sqlInsertPersona   = `INSERT INTO personas (user_id, name) VALUES ($1, $2) RETURNING id`
	sqlInsertPost      = `INSERT INTO posts (persona_id, title, text) VALUES ($1, $2, $3) RETURNING id`

	sqlUpdatePost = `UPDATE posts SET text = $1, updated_at = $2 WHERE id = $3`

	sqlRemovePersona = `DELETE FROM personas WHERE name = $1`
	sqlRemovePost    = `DELETE FROM posts WHERE id = $1`
)

type PsqlDB struct {
	db *sql.DB
}

func NewDB(databaseUrl string) *PsqlDB {
	var err error
	log.Printf("Connecting to postgres: %s\n", databaseUrl)

	db, err := sql.Open("postgres", databaseUrl)
	if err != nil {
		log.Fatal(err)
	}
	d := &PsqlDB{db: db}
	return d
}

func (me *PsqlDB) LinkUserKey(user *db.User, key string) error {
	_, err := me.db.Exec(sqlInsertPublicKey, user.ID, key)
	return err
}

func (me *PsqlDB) PublicKeyForKey(key string) (*db.PublicKey, error) {
	pk := &db.PublicKey{}
	r := me.db.QueryRow(sqlSelectPublicKey, key)
	err := r.Scan(&pk.ID, &pk.UserID, &pk.Key, &pk.CreatedAt)
	if err != nil {
		return pk, err
	}
	return pk, nil
}

func (me *PsqlDB) ListKeysForUser(user *db.User) ([]*db.PublicKey, error) {
	var keys []*db.PublicKey
	rs, err := me.db.Query(sqlSelectPublicKeys, user.ID)
	for rs.Next() {
		pk := &db.PublicKey{}
		err := rs.Scan(&pk.ID, &pk.UserID, &pk.Key, &pk.CreatedAt)
		if err != nil {
			return keys, err
		}

		keys = append(keys, pk)
	}
	if err != nil {
		return keys, err
	}
	if rs.Err() != nil {
		return keys, rs.Err()
	}
	return keys, nil
}

func (me *PsqlDB) UserForKey(key string) (*db.User, error) {
	pk, err := me.PublicKeyForKey(key)
	if err != nil {
		return nil, err
	}

	user, err := me.User(pk.UserID)
	if err != nil {
		return nil, err
	}

	personas, err := me.ListPersonas(user.ID)
	if err != nil {
		return nil, err
	}
	user.Personas = personas
	return user, nil
}

func (me *PsqlDB) User(userID string) (*db.User, error) {
	user := &db.User{}
	r := me.db.QueryRow(sqlSelectUser, userID)
	err := r.Scan(&user.ID, &user.CreatedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (me *PsqlDB) ListPersonas(userID string) ([]string, error) {
	var personas []string
	rs, err := me.db.Query(sqlSelectPersonas, userID)
	for rs.Next() {
		var name string
		err := rs.Scan(&name)
		if err != nil {
			return personas, err
		}

		personas = append(personas, name)
	}
	if err != nil {
		return personas, err
	}
	if rs.Err() != nil {
		return personas, rs.Err()
	}
	return personas, nil
}

func (me *PsqlDB) AddPersona(userID string, persona string) (string, error) {
	var id string
	err := me.db.QueryRow(sqlInsertPersona, userID, persona).Scan(&id)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (me *PsqlDB) RemovePersona(persona string) error {
	_, err := me.db.Exec(sqlRemovePersona, persona)
	return err
}

func (me *PsqlDB) FindPostWithTitle(title string, persona_id string) (*db.Post, error) {
	post := &db.Post{}
	r := me.db.QueryRow(sqlSelectPostWithTitle, title, persona_id)
	err := r.Scan(&post.ID, &post.PersonaID, &post.Title, &post.Text, &post.PublishAt)
	if err != nil {
		return nil, err
	}

	return post, nil
}

func (me *PsqlDB) FindPost(postID string) (*db.Post, error) {
	post := &db.Post{}
	r := me.db.QueryRow(sqlSelectPost, postID)
	err := r.Scan(&post.ID, &post.PersonaID, &post.Title, &post.Text, &post.PublishAt)
	if err != nil {
		return nil, err
	}

	return post, nil
}

func (me *PsqlDB) InsertPost(personaID string, title string, text string) (*db.Post, error) {
	var id string
	err := me.db.QueryRow(sqlInsertPost, personaID, title, text).Scan(&id)
	if err != nil {
		return nil, err
	}

	return me.FindPost(id)
}

func (me *PsqlDB) UpdatePost(postID string, text string) (*db.Post, error) {
	_, err := me.db.Exec(sqlInsertPost, text, postID)
	if err != nil {
		return nil, err
	}

	return me.FindPost(postID)
}

func (me *PsqlDB) RemovePost(postID string) error {
	_, err := me.db.Exec(sqlRemovePost, postID)
	return err
}

func (me *PsqlDB) Close() error {
	log.Println("Closing db")
	return me.db.Close()
}
