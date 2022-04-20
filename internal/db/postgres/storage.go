package postgres

import (
	"database/sql"
	"errors"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/neurosnap/lists.sh/internal/db"
)

const (
	sqlSelectPublicKey   = `SELECT id, user_id, public_key, created_at FROM public_keys WHERE public_key = $1`
	sqlSelectPublicKeys  = `SELECT id, user_id, public_key, created_at FROM public_keys WHERE user_id = $1`
	sqlSelectUser        = `SELECT id, name, created_at FROM app_users WHERE id = $1`
	sqlSelectUserForName = `SELECT id FROM app_users WHERE name = $1`

	sqlSelectPostWithTitle = `SELECT posts.id, user_id, title, text, publish_at, app_users.name as username FROM posts LEFT OUTER JOIN app_users ON app_users.id = posts.user_id WHERE title = $1 AND user_id = $2`
	sqlSelectPost          = `SELECT posts.id, user_id, title, text, publish_at, app_users.name as username FROM posts LEFT OUTER JOIN app_users ON app_users.id = posts.user_id WHERE posts.id = $1`
	sqlSelectPostsForUser  = `SELECT posts.id, user_id, title, text, publish_at, app_users.name as username FROM posts LEFT OUTER JOIN app_users ON app_users.id = posts.user_id WHERE user_id = $1`
	sqlSelectAllPosts      = `SELECT posts.id, user_id, title, text, publish_at, app_users.name as username FROM posts LEFT OUTER JOIN app_users ON app_users.id = posts.user_id ORDER BY publish_at DESC LIMIT 10 OFFSET $1`

	sqlInsertPublicKey = `INSERT INTO public_keys (user_id, public_key) VALUES ($1, $2)`
	sqlInsertPost      = `INSERT INTO posts (user_id, title, text, publish_at) VALUES ($1, $2, $3, $4) RETURNING id`
	sqlInsertUser      = `INSERT INTO app_users DEFAULT VALUES returning id`

	sqlUpdatePost     = `UPDATE posts SET text = $1, updated_at = $2, publish_at = $3 WHERE id = $4`
	sqlUpdateUserName = `UPDATE app_users SET name = $1 WHERE id = $2`

	sqlRemovePosts = `DELETE FROM posts WHERE id IN ($1)`
)

type PsqlDB struct {
	db *sql.DB
}

func NewDB() *PsqlDB {
	databaseUrl := os.Getenv("DATABASE_URL")
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

func (me *PsqlDB) LinkUserKey(userID string, key string) error {
	_, err := me.db.Exec(sqlInsertPublicKey, userID, key)
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

	return user, nil
}

func (me *PsqlDB) User(userID string) (*db.User, error) {
	user := &db.User{}
	var un sql.NullString
	r := me.db.QueryRow(sqlSelectUser, userID)
	err := r.Scan(&user.ID, &un, &user.CreatedAt)
	if err != nil {
		return nil, err
	}
	if un.Valid {
		user.Name = un.String
	}
	return user, nil
}

func (me *PsqlDB) ValidateName(name string) bool {
	userID, _ := me.UserForName(name)
	return userID == ""
}

func (me *PsqlDB) UserForName(name string) (string, error) {
	var id string
	r := me.db.QueryRow(sqlSelectUserForName, name)
	err := r.Scan(&id)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (me *PsqlDB) SetUserName(userID string, name string) error {
	if !me.ValidateName(name) {
		return errors.New("name is already taken")
	}

	_, err := me.db.Exec(sqlUpdateUserName, name, userID)
	return err
}

func (me *PsqlDB) FindPostWithTitle(title string, persona_id string) (*db.Post, error) {
	post := &db.Post{}
	r := me.db.QueryRow(sqlSelectPostWithTitle, title, persona_id)
	err := r.Scan(&post.ID, &post.UserID, &post.Title, &post.Text, &post.PublishAt, &post.Username)
	if err != nil {
		return nil, err
	}

	return post, nil
}

func (me *PsqlDB) FindPost(postID string) (*db.Post, error) {
	post := &db.Post{}
	r := me.db.QueryRow(sqlSelectPost, postID)
	err := r.Scan(&post.ID, &post.UserID, &post.Title, &post.Text, &post.PublishAt, &post.Username)
	if err != nil {
		return nil, err
	}

	return post, nil
}

func (me *PsqlDB) FindAllPosts(offset int) ([]*db.Post, error) {
	var posts []*db.Post
	rs, err := me.db.Query(sqlSelectAllPosts, offset)
	for rs.Next() {
		post := &db.Post{}
		err := rs.Scan(&post.ID, &post.UserID, &post.Title, &post.Text, &post.PublishAt, &post.Username)
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

func (me *PsqlDB) InsertPost(userID string, title string, text string, publishAt *time.Time) (*db.Post, error) {
	var id string
	err := me.db.QueryRow(sqlInsertPost, userID, title, text, publishAt).Scan(&id)
	if err != nil {
		return nil, err
	}

	return me.FindPost(id)
}

func (me *PsqlDB) UpdatePost(postID string, text string, publishAt *time.Time) (*db.Post, error) {
	_, err := me.db.Exec(sqlUpdatePost, text, time.Now(), publishAt, postID)
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
		err := rs.Scan(&post.ID, &post.UserID, &post.Title, &post.Text, &post.PublishAt, &post.Username)
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
