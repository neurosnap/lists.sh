package postgres

import (
	"database/sql"
	"errors"
	"math"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/neurosnap/lists.sh/internal"
	"github.com/neurosnap/lists.sh/internal/db"
	"golang.org/x/exp/slices"
)

var PAGER_SIZE = 15

const (
	sqlSelectPublicKey         = `SELECT id, user_id, public_key, created_at FROM public_keys WHERE public_key = $1`
	sqlSelectPublicKeys        = `SELECT id, user_id, public_key, created_at FROM public_keys WHERE user_id = $1`
	sqlSelectUser              = `SELECT id, name, created_at FROM app_users WHERE id = $1`
	sqlSelectUserForName       = `SELECT id, name, created_at FROM app_users WHERE name = $1`
	sqlSelectUserForNameAndKey = `SELECT app_users.id, app_users.name, app_users.created_at, public_keys.id as pk_id, public_keys.public_key, public_keys.created_at as pk_created_at FROM app_users LEFT OUTER JOIN public_keys ON public_keys.user_id = app_users.id WHERE app_users.name = $1 AND public_keys.public_key = $2`

	sqlSelectTotalUsers     = `SELECT count(id) FROM app_users`
	sqlSelectUsersLastMonth = `SELECT count(id) FROM app_users WHERE created_at >= $1`
	sqlSelectTotalPosts     = `SELECT count(id) FROM posts`
	sqlSelectPostsLastMonth = `SELECT count(id) FROM posts WHERE created_at >= $1`
	sqlSelectUsersWithPost  = `SELECT count(app_users.id) FROM app_users WHERE EXISTS (SELECT 1 FROM posts WHERE user_id = app_users.id);`

	sqlSelectPostWithFilename = `SELECT posts.id, user_id, filename, title, text, description, publish_at, app_users.name as username FROM posts LEFT OUTER JOIN app_users ON app_users.id = posts.user_id WHERE filename = $1 AND user_id = $2`
	sqlSelectPost             = `SELECT posts.id, user_id, filename, title, text, description, publish_at, app_users.name as username FROM posts LEFT OUTER JOIN app_users ON app_users.id = posts.user_id WHERE posts.id = $1`
	sqlSelectPostsForUser     = `SELECT posts.id, user_id, filename, title, text, description, publish_at, app_users.name as username FROM posts LEFT OUTER JOIN app_users ON app_users.id = posts.user_id WHERE user_id = $1 AND publish_at::date <= CURRENT_DATE ORDER BY publish_at DESC`
	sqlSelectAllPosts         = `SELECT posts.id, user_id, filename, title, text, description, publish_at, app_users.name as username FROM posts LEFT OUTER JOIN app_users ON app_users.id = posts.user_id WHERE filename <> '_readme' AND filename <> '_header' AND publish_at::date <= CURRENT_DATE ORDER BY publish_at DESC LIMIT $1 OFFSET $2`
	sqlSelectPostCount        = `SELECT count(id) FROM posts WHERE filename <> '_readme' AND filename <> '_header'`

	sqlInsertPublicKey = `INSERT INTO public_keys (user_id, public_key) VALUES ($1, $2)`
	sqlInsertPost      = `INSERT INTO posts (user_id, filename, title, text, description, publish_at) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`
	sqlInsertUser      = `INSERT INTO app_users DEFAULT VALUES returning id`

	sqlUpdatePost     = `UPDATE posts SET title = $1, text = $2, description = $3, updated_at = $4, publish_at = $5 WHERE id = $6`
	sqlUpdateUserName = `UPDATE app_users SET name = $1 WHERE id = $2`

	sqlRemovePosts = `DELETE FROM posts WHERE id IN ($1)`
)

type PsqlDB struct {
	db *sql.DB
}

func NewDB() *PsqlDB {
	databaseUrl := os.Getenv("DATABASE_URL")
	var err error
	logger := internal.CreateLogger()
	logger.Infof("Connecting to postgres: %s", databaseUrl)

	db, err := sql.Open("postgres", databaseUrl)
	if err != nil {
		logger.Fatal(err)
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
	var keys []*db.PublicKey
	rs, err := me.db.Query(sqlSelectPublicKey, key)
	if err != nil {
		return nil, err
	}

	for rs.Next() {
		pk := &db.PublicKey{}
		err := rs.Scan(&pk.ID, &pk.UserID, &pk.Key, &pk.CreatedAt)
		if err != nil {
			return nil, err
		}

		keys = append(keys, pk)
	}

	if rs.Err() != nil {
		return nil, rs.Err()
	}

	if len(keys) == 0 {
		return nil, errors.New("no public keys found for key provided")
	}

	// When we run PublicKeyForKey and there are multiple public keys returned from the database
	// that should mean that we don't have the correct username for this public key.
	// When that happens we need to reject the authentication and ask the user to provide the correct
	// username when using ssh.  So instead of `ssh lists.sh` it should be `ssh user@lists.sh`
	if len(keys) > 1 {
		return nil, &db.ErrMultiplePublicKeys{}
	}

	return keys[0], nil
}

func (me *PsqlDB) ListKeysForUser(user *db.User) ([]*db.PublicKey, error) {
	var keys []*db.PublicKey
	rs, err := me.db.Query(sqlSelectPublicKeys, user.ID)
	if err != nil {
		return keys, err
	}
	for rs.Next() {
		pk := &db.PublicKey{}
		err := rs.Scan(&pk.ID, &pk.UserID, &pk.Key, &pk.CreatedAt)
		if err != nil {
			return keys, err
		}

		keys = append(keys, pk)
	}
	if rs.Err() != nil {
		return keys, rs.Err()
	}
	return keys, nil
}

func (me *PsqlDB) SiteAnalytics() (*db.Analytics, error) {
	analytics := &db.Analytics{}
	r := me.db.QueryRow(sqlSelectTotalUsers)
	err := r.Scan(&analytics.TotalUsers)
	if err != nil {
		return nil, err
	}

	r = me.db.QueryRow(sqlSelectTotalPosts)
	err = r.Scan(&analytics.TotalPosts)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	year, month, _ := now.Date()
	lastMonth := time.Date(year, month-1, 1, 0, 0, 0, 0, now.Location())

	r = me.db.QueryRow(sqlSelectPostsLastMonth, lastMonth)
	err = r.Scan(&analytics.PostsLastMonth)
	if err != nil {
		return nil, err
	}

	r = me.db.QueryRow(sqlSelectUsersLastMonth, lastMonth)
	err = r.Scan(&analytics.UsersLastMonth)
	if err != nil {
		return nil, err
	}

	r = me.db.QueryRow(sqlSelectUsersWithPost)
	err = r.Scan(&analytics.UsersWithPost)
	if err != nil {
		return nil, err
	}

	return analytics, nil
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
	lower := strings.ToLower(name)
	if slices.Contains(db.DenyList, lower) {
		return false
	}
	v := db.NameValidator.MatchString(lower)
	if !v {
		return false
	}
	user, _ := me.UserForName(lower)
	return user == nil
}

func (me *PsqlDB) UserForName(name string) (*db.User, error) {
	user := &db.User{}
	r := me.db.QueryRow(sqlSelectUserForName, strings.ToLower(name))
	err := r.Scan(&user.ID, &user.Name, &user.CreatedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (me *PsqlDB) UserForNameAndKey(name string, key string) (*db.User, error) {
	user := &db.User{}
	pk := &db.PublicKey{}

	r := me.db.QueryRow(sqlSelectUserForNameAndKey, strings.ToLower(name), key)
	err := r.Scan(&user.ID, &user.Name, &user.CreatedAt, &pk.ID, &pk.Key, &pk.CreatedAt)
	if err != nil {
		return nil, err
	}

	user.PublicKey = pk
	return user, nil
}

func (me *PsqlDB) SetUserName(userID string, name string) error {
	lowerName := strings.ToLower(name)
	if !me.ValidateName(lowerName) {
		return errors.New("name is already taken")
	}

	_, err := me.db.Exec(sqlUpdateUserName, lowerName, userID)
	return err
}

func (me *PsqlDB) FindPostWithFilename(filename string, persona_id string) (*db.Post, error) {
	post := &db.Post{}
	r := me.db.QueryRow(sqlSelectPostWithFilename, filename, persona_id)
	err := r.Scan(
		&post.ID,
		&post.UserID,
		&post.Filename,
		&post.Title,
		&post.Text,
		&post.Description,
		&post.PublishAt,
		&post.Username,
	)
	if err != nil {
		return nil, err
	}

	return post, nil
}

func (me *PsqlDB) FindPost(postID string) (*db.Post, error) {
	post := &db.Post{}
	r := me.db.QueryRow(sqlSelectPost, postID)
	err := r.Scan(
		&post.ID,
		&post.UserID,
		&post.Filename,
		&post.Title,
		&post.Text,
		&post.Description,
		&post.PublishAt,
		&post.Username,
	)
	if err != nil {
		return nil, err
	}

	return post, nil
}

func (me *PsqlDB) FindAllPosts(page *db.Pager) (*db.Paginate[*db.Post], error) {
	var posts []*db.Post
	rs, err := me.db.Query(sqlSelectAllPosts, page.Num, page.Num*page.Page)
	if err != nil {
		return nil, err
	}
	for rs.Next() {
		post := &db.Post{}
		err := rs.Scan(
			&post.ID,
			&post.UserID,
			&post.Filename,
			&post.Title,
			&post.Text,
			&post.Description,
			&post.PublishAt,
			&post.Username,
		)
		if err != nil {
			return nil, err
		}

		posts = append(posts, post)
	}
	if rs.Err() != nil {
		return nil, rs.Err()
	}

	var count int
	err = me.db.QueryRow(sqlSelectPostCount).Scan(&count)
	if err != nil {
		return nil, err
	}

	pager := &db.Paginate[*db.Post]{
		Data:  posts,
		Total: int(math.Ceil(float64(count) / float64(page.Num))),
	}
	return pager, nil
}

func (me *PsqlDB) InsertPost(userID string, filename string, title string, text string, description string, publishAt *time.Time) (*db.Post, error) {
	var id string
	err := me.db.QueryRow(sqlInsertPost, userID, filename, title, text, description, publishAt).Scan(&id)
	if err != nil {
		return nil, err
	}

	return me.FindPost(id)
}

func (me *PsqlDB) UpdatePost(postID string, title string, text string, description string, publishAt *time.Time) (*db.Post, error) {
	_, err := me.db.Exec(sqlUpdatePost, title, text, description, time.Now(), publishAt, postID)
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
	if err != nil {
		return posts, err
	}
	for rs.Next() {
		post := &db.Post{}
		err := rs.Scan(
			&post.ID,
			&post.UserID,
			&post.Filename,
			&post.Title,
			&post.Text,
			&post.Description,
			&post.PublishAt,
			&post.Username,
		)
		if err != nil {
			return posts, err
		}

		posts = append(posts, post)
	}
	if rs.Err() != nil {
		return posts, rs.Err()
	}
	return posts, nil
}

func (me *PsqlDB) Close() error {
	logger := internal.CreateLogger()
	logger.Info("Closing db")
	return me.db.Close()
}
