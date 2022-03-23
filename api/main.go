package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

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

	for _, post := range syncReq.Posts {
		log.Println(post)
	}

	json.NewEncoder(w).Encode(map[string]string{"hi": "mom"})
}

func StartServer() {
	http.HandleFunc("/sync", sync)

	port := os.Getenv("PORT")
	portStr := fmt.Sprintf(":%s", port)
	log.Fatal(http.ListenAndServe(portStr, nil))
}
