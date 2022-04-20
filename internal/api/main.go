package api

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/neurosnap/lists.sh/internal"
	"github.com/neurosnap/lists.sh/internal/db/postgres"
	routeHelper "github.com/neurosnap/lists.sh/internal/router"
)

func renderTemplate(templates []string) (*template.Template, error) {
	files := make([]string, len(templates))
	copy(files, templates)
	files = append(
		files,
		"./html/footer.partial.tmpl",
		"./html/base.layout.tmpl",
	)

	ts, err := template.ParseFiles(files...)
	if err != nil {
		return nil, err
	}
	return ts, nil
}

func marketingHandler(w http.ResponseWriter, r *http.Request) {
	ts, err := renderTemplate([]string{
		"./html/marketing.page.tmpl",
	})

	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", 500)
		return
	}

	err = ts.Execute(w, nil)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", 500)
	}
}

type BlogData struct {
	PageTitle string
	Username  string
	Posts     []PostItemData
}

func blogHandler(w http.ResponseWriter, r *http.Request) {
	username := routeHelper.GetField(r, 0)
	dbpool := routeHelper.GetDB(r)
	userID, err := dbpool.UserForName(username)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", 500)
		return
	}
	posts, err := dbpool.PostsForUser(userID)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", 500)
		return
	}

	ts, err := renderTemplate([]string{
		"./html/blog.page.tmpl",
	})

	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", 500)
		return
	}

	postCollection := make([]PostItemData, 0, len(posts))
	for _, post := range posts {
		p := PostItemData{
			URL:         fmt.Sprintf("/%s/%s", post.Username, post.Title),
			Title:       internal.FilenameToTitle(post.Title),
			PublishedAt: post.PublishAt.Format("Mon January 2, 2006"),
		}
		postCollection = append(postCollection, p)
	}

	data := BlogData{
		PageTitle: fmt.Sprintf("%s -- lists.sh", username),
		Username:  username,
		Posts:     postCollection,
	}

	err = ts.Execute(w, data)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", 500)
	}
}

type ItemData struct {
	Text string
}

type PostData struct {
	PageTitle string
	Title     string
	Username  string
	Items     []ItemData
}

func postHandler(w http.ResponseWriter, r *http.Request) {
	username := routeHelper.GetField(r, 0)
	title := routeHelper.GetField(r, 1)
	dbpool := routeHelper.GetDB(r)
	userID, err := dbpool.UserForName(username)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", 500)
		return
	}
	post, err := dbpool.FindPostWithTitle(title, userID)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", 500)
		return
	}

	items := []ItemData{}
	itemsArr := strings.Split(post.Text, "\n")
	for _, text := range itemsArr {
		items = append(items, ItemData{Text: text})
	}
	data := PostData{
		PageTitle: post.Title,
		Title:     internal.FilenameToTitle(post.Title),
		Username:  username,
		Items:     items,
	}

	ts, err := renderTemplate([]string{
		"./html/post.page.tmpl",
	})

	if err != nil {
		http.Error(w, "Internal Server Error", 500)
	}

	err = ts.Execute(w, data)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", 500)
	}
}

type PostItemData struct {
	URL         string
	Title       string
	Username    string
	PublishedAt string
}

type ReadData struct {
	Posts []PostItemData
}

func readHandler(w http.ResponseWriter, r *http.Request) {
	dbpool := routeHelper.GetDB(r)
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	posts, err := dbpool.FindAllPosts(page)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", 500)
		return
	}

	ts, err := renderTemplate([]string{
		"./html/read.page.tmpl",
	})

	if err != nil {
		http.Error(w, "Internal Server Error", 500)
	}

	data := ReadData{}
	for _, post := range posts {
		item := PostItemData{
			URL:         fmt.Sprintf("/%s/%s", post.Username, post.Title),
			Title:       internal.FilenameToTitle(post.Title),
			Username:    post.Username,
			PublishedAt: post.PublishAt.Format("Mon January 2, 2006"),
		}
		data.Posts = append(data.Posts, item)
	}

	err = ts.Execute(w, data)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", 500)
	}
}

func serveFile(file string, contentType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		contents, err := os.ReadFile(fmt.Sprintf("./public/%s", file))
		if err != nil {
			log.Println(err)
			http.Error(w, "File not found", 404)
		}
		w.Header().Add("Content-Type", contentType)
		fmt.Fprintf(w, string(contents))
	}
}

var routes = []routeHelper.Route{
	routeHelper.NewRoute("GET", "/", marketingHandler),
	routeHelper.NewRoute("GET", "/main.css", serveFile("main.css", "text/css")),
	routeHelper.NewRoute("GET", "/read", readHandler),
	routeHelper.NewRoute("GET", "/([^/]+)", blogHandler),
	routeHelper.NewRoute("GET", "/([^/]+)/([^/]+)", postHandler),
}

func StartServer() {
	db := postgres.NewDB()
	defer db.Close()

	handler := routeHelper.CreateServe(routes, db)
	router := http.HandlerFunc(handler)

	port := internal.GetEnv("PORT", "3000")
	portStr := fmt.Sprintf(":%s", port)
	log.Printf("Starting server on port %s", port)
	log.Fatal(http.ListenAndServe(portStr, router))
}
