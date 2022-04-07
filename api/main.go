package api

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/neurosnap/lists.sh/internal"
)

type RequestDBKey string

func appHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello world!")
}

func StartServer() {
	databaseUrl := os.Getenv("DATABASE_URL")
	log.Println(databaseUrl)

	http.HandleFunc("/", appHandler)

	port := internal.GetEnv("PORT", "3000")
	portStr := fmt.Sprintf(":%s", port)
	log.Printf("Starting server on port %s", port)
	log.Fatal(http.ListenAndServe(portStr, nil))
}
