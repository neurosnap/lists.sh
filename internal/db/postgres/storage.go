package postgres

import (
	"database/sql"
	"log"
	"strings"

	_ "github.com/lib/pq"
	"github.com/neurosnap/lists.sh/internal/db"
)

const (
	sqlSelectPublicKey      = `SELECT id, user_id, public_key, created_at FROM public_keys WHERE public_key = $1`
	sqlSelectPublicKeys     = `SELECT id, user_id, public_key, created_at FROM public_keys WHERE user_id = $1`
	sqlSelectUser           = `SELECT id, created_at FROM app_users WHERE id = $1`
	sqlSelectPersonaForName = `SELECT id FROM personas WHERE name = $1`
	sqlSelectPersona        = `SELECT id, name, created_at FROM personas where id = $1`
	sqlSelectPersonas       = `SELECT id, name, created_at FROM personas WHERE user_id = $1`
	sqlSelectPostWithTitle  = `SELECT id, persona_id, title, text, publish_at FROM posts WHERE title = $1 AND persona_id = $2`
	sqlSelectPost           = `SELECT id, persona_id, title, text, publish_at FROM posts WHERE id = $1`
	sqlSelectPostsForUser   = `SELECT posts.id, persona_id, title, text, publish_at FROM posts LEFT OUTER JOIN personas ON personas.id = posts.persona_id WHERE personas.user_id = $1`

	sqlInsertPublicKey = `INSERT INTO public_keys (user_id, public_key) VALUES ($1, $2)`
	sqlInsertPersona   = `INSERT INTO personas (user_id, name) VALUES ($1, $2) RETURNING id`
	sqlInsertPost      = `INSERT INTO posts (persona_id, title, text) VALUES ($1, $2, $3) RETURNING id`
	sqlInsertUser      = `INSERT INTO app_users DEFAULT VALUES returning id`

	sqlUpdatePost = `UPDATE posts SET text = $1, updated_at = $2 WHERE id = $3`

	sqlRemovePersona = `DELETE FROM personas WHERE name = $1`
	sqlRemovePosts    = `DELETE FROM posts WHERE id IN ($1)`
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

func (me *PsqlDB) AddUser() (string, error) {
	var id string
	err := me.db.QueryRow(sqlInsertUser).Scan(&id)
	if err != nil {
		return "", err
	}
	return id, nil
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

	user.PublicKey = pk

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

func (me *PsqlDB) ValidateName(name string) bool {
	var id string
	r := me.db.QueryRow(sqlSelectPersonaForName, name)
	err := r.Scan(&id)
	if err != nil {
		return true
	}
	return id == ""
}

func (me *PsqlDB) ListPersonas(userID string) ([]*db.Persona, error) {
	var personas []*db.Persona
	rs, err := me.db.Query(sqlSelectPersonas, userID)
	for rs.Next() {
		persona := &db.Persona{}
		err := rs.Scan(&persona.ID, &persona.Name, &persona.CreatedAt)
		if err != nil {
			return personas, err
		}

		personas = append(personas, persona)
	}
	if err != nil {
		return personas, err
	}
	if rs.Err() != nil {
		return personas, rs.Err()
	}
	return personas, nil
}

func (me *PsqlDB) FindPersona(personaID string) (*db.Persona, error) {
	persona := &db.Persona{}
	err := me.db.QueryRow(sqlSelectPersona, personaID).Scan(&persona.ID, &persona.Name, &persona.CreatedAt)
	return persona, err
}

func (me *PsqlDB) AddPersona(userID string, persona string) (*db.Persona, error) {
	if !me.ValidateName(persona) {
		return nil, db.ErrNameTaken
	}
	var id string
	err := me.db.QueryRow(sqlInsertPersona, userID, persona).Scan(&id)
	if err != nil {
		return nil, err
	}
	return me.FindPersona(id)
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

func (me *PsqlDB) RemovePosts(postIDs []string) error {
	_, err := me.db.Exec(sqlRemovePosts, strings.Join(postIDs, ","))
	return err
}

func (me *PsqlDB) PostsForUser(userID string) ([]*db.Post, error) {
	var posts []*db.Post
	rs, err := me.db.Query(sqlSelectPostsForUser, userID)
	for rs.Next() {
		post := &db.Post{}
		err := rs.Scan(&post.ID, &post.PersonaID, &post.Title, &post.Text, &post.PublishAt)
		if err != nil {
			return posts, err
		}

		posts = append(posts, post)
	}
	if err != nil {
		return posts, err
	}
	if rs.Err() != nil {
		return posts, rs.Err()
	}
	return posts, nil
}

func (me *PsqlDB) Close() error {
	log.Println("Closing db")
	return me.db.Close()
}
