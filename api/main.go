package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/neurosnap/lists.sh/internal"
)

type RequestDBKey string

type dbconn interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, optionsAndArgs ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, optionsAndArgs ...interface{}) pgx.Row
}

type PostRequest struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

type SyncRequest struct {
	Posts []PostRequest `json:"posts"`
}

func sync(w http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	var syncReq SyncRequest
	err := decoder.Decode(&syncReq)
	if err != nil {
		panic(err)
	}

    ctx := req.Context()
	db := ctx.Value(RequestDBKey("db")).(dbconn)

    titleMap := make(map[string]int)
    for _, post := range syncReq.Posts {
        titleMap[post.Title] = 1
    }

    rows, _ := db.Query(ctx, "SELECT title FROM posts")
    for rows.Next() {
        var title string
        rows.Scan(&title)
        log.Println(title)
        titleMap[title] -= 1
    }
    if rows.Err() != nil {
        log.Println("ERROR")
    }
    log.Println(titleMap)

	json.NewEncoder(w).Encode(titleMap)
}

func pgxPoolHandler(dbpool *pgxpool.Pool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			ctx = context.WithValue(ctx, RequestDBKey("db"), dbpool)
			next.ServeHTTP(w, r.WithContext(ctx))
		}

		return http.HandlerFunc(fn)
	}
}

func StartServer() {
	mux := http.NewServeMux()
	databaseUrl := os.Getenv("DATABASE_URL")
	dbpool, err := pgxpool.Connect(context.Background(), databaseUrl)
	if err != nil {
		log.Fatal("failed to connect to database")
		return
	}
	mdw := pgxPoolHandler(dbpool)

	syncHandler := http.HandlerFunc(sync)
	mux.Handle("/sync", mdw(syncHandler))

	port := internal.GetEnv("PORT", "3000")
	portStr := fmt.Sprintf(":%s", port)
	log.Printf("Starting server on port %s", port)
	log.Fatal(http.ListenAndServe(portStr, mux))
}
