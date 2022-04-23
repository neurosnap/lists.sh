package api

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/feeds"
	"github.com/neurosnap/lists.sh/internal"
	"github.com/neurosnap/lists.sh/internal/db"
	"github.com/neurosnap/lists.sh/internal/db/postgres"
	routeHelper "github.com/neurosnap/lists.sh/internal/router"
	"github.com/neurosnap/lists.sh/pkg"
)

func renderTemplate(templates []string) (*template.Template, error) {
	files := make([]string, len(templates))
	copy(files, templates)
	files = append(
		files,
		"./html/footer.partial.tmpl",
		"./html/marketing-footer.partial.tmpl",
		"./html/base.layout.tmpl",
	)

	ts, err := template.ParseFiles(files...)
	if err != nil {
		return nil, err
	}
	return ts, nil
}

func createPageHandler(fname string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ts, err := renderTemplate([]string{fname})

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
}

type BlogData struct {
	PageTitle string
	Bio       string
	Username  string
	Posts     []PostItemData
}

func getBlogTitle(user *db.User) string {
	if user.Bio == "" {
		return user.Name
	}

	return fmt.Sprintf("%s: %s", user.Name, user.Bio)
}

func blogHandler(w http.ResponseWriter, r *http.Request) {
	username := routeHelper.GetField(r, 0)
	dbpool := routeHelper.GetDB(r)
	user, err := dbpool.UserForName(username)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", 500)
		return
	}
	posts, err := dbpool.PostsForUser(user.ID)
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
			URL:       fmt.Sprintf("/%s/%s", post.Username, post.Filename),
			Title:     internal.FilenameToTitle(post.Filename, post.Title),
			PublishAt: post.PublishAt.Format("02 Jan, 2006"),
		}
		postCollection = append(postCollection, p)
	}

	data := BlogData{
		PageTitle: getBlogTitle(user),
		Bio:       user.Bio,
		Username:  username,
		Posts:     postCollection,
	}

	err = ts.Execute(w, data)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", 500)
	}
}

type PostData struct {
	PageTitle   string
	Title       string
	Description string
	Username    string
	PublishAt   string
	Items       []*pkg.ListItem
}

func getPostTitle(post *db.Post) string {
	return fmt.Sprintf("%s: %s", post.Title, post.Description)
}

func postHandler(w http.ResponseWriter, r *http.Request) {
	username := routeHelper.GetField(r, 0)
	filename := routeHelper.GetField(r, 1)
	dbpool := routeHelper.GetDB(r)
	user, err := dbpool.UserForName(username)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", 500)
		return
	}
	post, err := dbpool.FindPostWithFilename(filename, user.ID)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", 500)
		return
	}

	parsedText := pkg.ParseText(post.Text)

	data := PostData{
		PageTitle:   getPostTitle(post),
		Description: post.Description,
		Title:       internal.FilenameToTitle(post.Filename, post.Title),
		PublishAt:   post.PublishAt.Format("Mon January 2, 2006"),
		Username:    username,
		Items:       parsedText.Items,
	}

	ts, err := renderTemplate([]string{
		"./html/post.page.tmpl",
		"./html/list.partial.tmpl",
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
	Description string
	Username    string
	PublishAt   string
}

type ReadData struct {
	Posts    []PostItemData
	NextPage string
	PrevPage string
}

func readHandler(w http.ResponseWriter, r *http.Request) {
	dbpool := routeHelper.GetDB(r)
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pager, err := dbpool.FindAllPosts(page)
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

	nextPage := ""
	if page < pager.Total-1 {
		nextPage = fmt.Sprintf("/read?page=%d", page+1)
	}

	prevPage := ""
	if page > 0 {
		prevPage = fmt.Sprintf("/read?page=%d", page-1)
	}

	data := ReadData{
		NextPage: nextPage,
		PrevPage: prevPage,
	}
	for _, post := range pager.Data {
		item := PostItemData{
			URL:         fmt.Sprintf("/%s/%s", post.Username, post.Filename),
			Title:       internal.FilenameToTitle(post.Filename, post.Title),
			Description: post.Description,
			Username:    post.Username,
			PublishAt:   post.PublishAt.Format("02 Jan, 2006"),
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

func rssHandler(w http.ResponseWriter, r *http.Request) {
	username := routeHelper.GetField(r, 0)
	dbpool := routeHelper.GetDB(r)
	user, err := dbpool.UserForName(username)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", 500)
		return
	}
	posts, err := dbpool.PostsForUser(user.ID)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", 500)
		return
	}

	ts, err := template.ParseFiles("./html/rss.page.tmpl", "./html/list.partial.tmpl")
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", 500)
		return
	}

	feed := &feeds.Feed{
		Title:       fmt.Sprintf("%s's blog", username),
		Link:        &feeds.Link{Href: fmt.Sprintf("https://lists.sh/%s/rss", username)},
		Description: user.Bio,
		Author:      &feeds.Author{Name: username},
		Created:     time.Now(),
	}

	var feedItems []*feeds.Item
	for _, post := range posts {
		parsed := pkg.ParseText(post.Text)
		var tpl bytes.Buffer
		data := &PostData{Items: parsed.Items}
		if err := ts.Execute(&tpl, data); err != nil {
			continue
		}
		feedItems = append(feedItems, &feeds.Item{
			Id:          post.ID,
			Title:       post.Title,
			Link:        &feeds.Link{Href: fmt.Sprintf("https://lists.sh/%s/%s", username, post.Title)},
			Description: post.Description,
			Content:     tpl.String(),
			Created:     *post.PublishAt,
		})
	}
	feed.Items = feedItems

	rss, err := feed.ToAtom()
	if err != nil {
		log.Fatal(err)
		http.Error(w, "Could not generate atom rss feed", 500)
	}

	w.Header().Add("Content-Type", "application/atom+xml")
	fmt.Fprintf(w, rss)
}

var routes = []routeHelper.Route{
	routeHelper.NewRoute("GET", "/", createPageHandler("./html/marketing.page.tmpl")),
	routeHelper.NewRoute("GET", "/spec", createPageHandler("./html/spec.page.tmpl")),
	routeHelper.NewRoute("GET", "/ops", createPageHandler("./html/ops.page.tmpl")),
	routeHelper.NewRoute("GET", "/privacy", createPageHandler("./html/privacy.page.tmpl")),
	routeHelper.NewRoute("GET", "/transparency", createPageHandler("./html/transparency.page.tmpl")),
	routeHelper.NewRoute("GET", "/help", createPageHandler("./html/help.page.tmpl")),
	routeHelper.NewRoute("GET", "/main.css", serveFile("main.css", "text/css")),
	routeHelper.NewRoute("GET", "/read", readHandler),
	routeHelper.NewRoute("GET", "/([^/]+)", blogHandler),
	routeHelper.NewRoute("GET", "/([^/]+)/rss", rssHandler),
	routeHelper.NewRoute("GET", "/([^/]+)/([^/]+)", postHandler),
}

func StartServer() {
	db := postgres.NewDB()
	defer db.Close()

	handler := routeHelper.CreateServe(routes, db)
	router := http.HandlerFunc(handler)

	port := internal.GetEnv("LISTS_WEB_PORT", "3000")
	portStr := fmt.Sprintf(":%s", port)
	log.Printf("Starting server on port %s", port)
	log.Fatal(http.ListenAndServe(portStr, router))
}
