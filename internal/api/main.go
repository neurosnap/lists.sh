package api

import (
	"fmt"
	"log"
	"net/http"

	"github.com/neurosnap/lists.sh/internal"
	"github.com/neurosnap/lists.sh/internal/db/postgres"
)

type RequestDBKey string

func appHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello world!")
}

func StartServer() {
	db := postgres.NewDB()
	log.Println(db)

	http.HandleFunc("/", appHandler)

	port := internal.GetEnv("PORT", "3000")
	portStr := fmt.Sprintf(":%s", port)
	log.Printf("Starting server on port %s", port)
	log.Fatal(http.ListenAndServe(portStr, nil))
}
