package api

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/feeds"
	"github.com/neurosnap/lists.sh/internal"
	"github.com/neurosnap/lists.sh/internal/db"
	"github.com/neurosnap/lists.sh/internal/db/postgres"
	routeHelper "github.com/neurosnap/lists.sh/internal/router"
	"github.com/neurosnap/lists.sh/pkg"
)

type PostItemData struct {
	URL          string
	Username     string
	Title        string
	Description  string
	PublishAtISO string
	PublishAt    string
}

type BlogPageData struct {
	PageTitle string
	URL       string
	Username  string
	Readme    *ReadmeTxt
	Header    *HeaderTxt
	Posts     []PostItemData
}

type ReadPageData struct {
	NextPage string
	PrevPage string
	Posts    []PostItemData
}

type PostPageData struct {
	PageTitle    string
	URL          string
	Title        string
	Description  string
	Username     string
	BlogName     string
	ListType     string
	Items        []*pkg.ListItem
	PublishAtISO string
	PublishAt    string
}

func postURL(post *db.Post) string {
	return fmt.Sprintf("https://lists.sh/%s/%s", post.Username, post.Filename)
}

func blogURL(username string) string {
	return fmt.Sprintf("https://lists.sh/%s", username)
}

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
		logger := routeHelper.GetLogger(r)
		ts, err := renderTemplate([]string{fname})

		if err != nil {
			logger.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		err = ts.Execute(w, nil)
		if err != nil {
			logger.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

type HeaderTxt struct {
	Title    string
	Bio      string
	Nav      []*pkg.ListItem
	HasItems bool
}

type ReadmeTxt struct {
	HasItems bool
	ListType string
	Items    []*pkg.ListItem
}

func blogHandler(w http.ResponseWriter, r *http.Request) {
	username := routeHelper.GetField(r, 0)
	dbpool := routeHelper.GetDB(r)
	logger := routeHelper.GetLogger(r)

	user, err := dbpool.UserForName(username)
	if err != nil {
		logger.Infof("blog not found: %s", username)
		http.Error(w, "blog not found", http.StatusNotFound)
		return
	}
	posts, err := dbpool.PostsForUser(user.ID)
	if err != nil {
		logger.Error(err)
		http.Error(w, "could not fetch posts for blog", http.StatusInternalServerError)
		return
	}

	ts, err := renderTemplate([]string{
		"./html/blog.page.tmpl",
		"./html/list.partial.tmpl",
	})

	if err != nil {
		logger.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	headerTxt := &HeaderTxt{
		Title: fmt.Sprintf("%s's blog", username),
		Bio:   "",
	}
	readmeTxt := &ReadmeTxt{}

	postCollection := make([]PostItemData, 0, len(posts))
	for _, post := range posts {
		if post.Filename == "_header" {
			parsedText := pkg.ParseText(post.Text)
			if parsedText.MetaData.Title != "" {
				headerTxt.Title = parsedText.MetaData.Title
			}

			if parsedText.MetaData.Description != "" {
				headerTxt.Bio = parsedText.MetaData.Description
			}

			headerTxt.Nav = parsedText.Items
			if len(headerTxt.Nav) > 0 {
				headerTxt.HasItems = true
			}
		} else if post.Filename == "_readme" {
			parsedText := pkg.ParseText(post.Text)
			readmeTxt.Items = parsedText.Items
			readmeTxt.ListType = parsedText.MetaData.ListType
			if len(readmeTxt.Items) > 0 {
				readmeTxt.HasItems = true
			}
		} else {
			p := PostItemData{
				URL:          fmt.Sprintf("/%s/%s", post.Username, post.Filename),
				Title:        internal.FilenameToTitle(post.Filename, post.Title),
				PublishAt:    post.PublishAt.Format("02 Jan, 2006"),
				PublishAtISO: post.PublishAt.Format(time.RFC3339),
			}
			postCollection = append(postCollection, p)
		}
	}

	data := BlogPageData{
		PageTitle: headerTxt.Title,
		URL:       blogURL(username),
		Readme:    readmeTxt,
		Header:    headerTxt,
		Username:  username,
		Posts:     postCollection,
	}

	err = ts.Execute(w, data)
	if err != nil {
		logger.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func getPostTitle(post *db.Post) string {
	if post.Description == "" {
		return post.Title
	}

	return fmt.Sprintf("%s: %s", post.Title, post.Description)
}

func getBlogName(username string) string {
	return fmt.Sprintf("%s's blog", username)
}

func postHandler(w http.ResponseWriter, r *http.Request) {
	username := routeHelper.GetField(r, 0)
	filename := routeHelper.GetField(r, 1)
	dbpool := routeHelper.GetDB(r)
	logger := routeHelper.GetLogger(r)

	user, err := dbpool.UserForName(username)
	if err != nil {
		logger.Infof("blog not found: %s", username)
		http.Error(w, "blog not found", http.StatusNotFound)
		return
	}

	header, _ := dbpool.FindPostWithFilename("_header", user.ID)
	blogName := getBlogName(username)
	if header != nil {
		headerParsed := pkg.ParseText(header.Text)
		if headerParsed.MetaData.Title != "" {
			blogName = headerParsed.MetaData.Title
		}
	}

	post, err := dbpool.FindPostWithFilename(filename, user.ID)
	if err != nil {
		logger.Infof("post not found %s/%s", username, filename)
		http.Error(w, "post not found", http.StatusNotFound)
		return
	}

	parsedText := pkg.ParseText(post.Text)

	data := PostPageData{
		PageTitle:    getPostTitle(post),
		URL:          postURL(post),
		Description:  post.Description,
		ListType:     parsedText.MetaData.ListType,
		Title:        internal.FilenameToTitle(post.Filename, post.Title),
		PublishAt:    post.PublishAt.Format("Mon January 2, 2006"),
		PublishAtISO: post.PublishAt.Format(time.RFC3339),
		Username:     username,
		BlogName:     blogName,
		Items:        parsedText.Items,
	}

	ts, err := renderTemplate([]string{
		"./html/post.page.tmpl",
		"./html/list.partial.tmpl",
	})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	err = ts.Execute(w, data)
	if err != nil {
		logger.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func transparencyHandler(w http.ResponseWriter, r *http.Request) {
	dbpool := routeHelper.GetDB(r)
	logger := routeHelper.GetLogger(r)

	analytics, err := dbpool.SiteAnalytics()
	if err != nil {
		logger.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ts, err := template.ParseFiles(
		"./html/transparency.page.tmpl",
		"./html/footer.partial.tmpl",
		"./html/marketing-footer.partial.tmpl",
		"./html/base.layout.tmpl",
	)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	err = ts.Execute(w, analytics)
	if err != nil {
		logger.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func readHandler(w http.ResponseWriter, r *http.Request) {
	dbpool := routeHelper.GetDB(r)
	logger := routeHelper.GetLogger(r)

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pager, err := dbpool.FindAllPosts(&db.Pager{Num: 20, Page: page})
	if err != nil {
		logger.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ts, err := renderTemplate([]string{
		"./html/read.page.tmpl",
	})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	nextPage := ""
	if page < pager.Total-1 {
		nextPage = fmt.Sprintf("/read?page=%d", page+1)
	}

	prevPage := ""
	if page > 0 {
		prevPage = fmt.Sprintf("/read?page=%d", page-1)
	}

	data := ReadPageData{
		NextPage: nextPage,
		PrevPage: prevPage,
	}
	for _, post := range pager.Data {
		item := PostItemData{
			URL:          fmt.Sprintf("/%s/%s", post.Username, post.Filename),
			Title:        internal.FilenameToTitle(post.Filename, post.Title),
			Description:  post.Description,
			Username:     post.Username,
			PublishAt:    post.PublishAt.Format("02 Jan, 2006"),
			PublishAtISO: post.PublishAt.Format(time.RFC3339),
		}
		data.Posts = append(data.Posts, item)
	}

	err = ts.Execute(w, data)
	if err != nil {
		logger.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func rssBlogHandler(w http.ResponseWriter, r *http.Request) {
	username := routeHelper.GetField(r, 0)
	dbpool := routeHelper.GetDB(r)
	logger := routeHelper.GetLogger(r)

	user, err := dbpool.UserForName(username)
	if err != nil {
		logger.Infof("rss feed not found: %s", username)
		http.Error(w, "rss feed not found", http.StatusNotFound)
		return
	}
	posts, err := dbpool.PostsForUser(user.ID)
	if err != nil {
		logger.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ts, err := template.ParseFiles("./html/rss.page.tmpl", "./html/list.partial.tmpl")
	if err != nil {
		logger.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	headerTxt := &HeaderTxt{
		Title: getBlogName(username),
	}

	for _, post := range posts {
		if post.Filename == "_header" {
			parsedText := pkg.ParseText(post.Text)
			if parsedText.MetaData.Title != "" {
				headerTxt.Title = parsedText.MetaData.Title
			}

			if parsedText.MetaData.Description != "" {
				headerTxt.Bio = parsedText.MetaData.Description
			}

			break
		}
	}

	feed := &feeds.Feed{
		Title:       headerTxt.Title,
		Link:        &feeds.Link{Href: blogURL(username)},
		Description: headerTxt.Bio,
		Author:      &feeds.Author{Name: username},
		Created:     time.Now(),
	}

	var feedItems []*feeds.Item
	for _, post := range posts {
		parsed := pkg.ParseText(post.Text)
		var tpl bytes.Buffer
		data := &PostPageData{
			ListType: parsed.MetaData.ListType,
			Items:    parsed.Items,
		}
		if err := ts.Execute(&tpl, data); err != nil {
			continue
		}
		feedItems = append(feedItems, &feeds.Item{
			Id:          post.ID,
			Title:       post.Title,
			Link:        &feeds.Link{Href: postURL(post)},
			Description: post.Description,
			Content:     tpl.String(),
			Created:     *post.PublishAt,
		})
	}
	feed.Items = feedItems

	rss, err := feed.ToAtom()
	if err != nil {
		logger.Fatal(err)
		http.Error(w, "Could not generate atom rss feed", http.StatusInternalServerError)
	}

	w.Header().Add("Content-Type", "application/atom+xml")
	fmt.Fprintf(w, rss)
}

func rssHandler(w http.ResponseWriter, r *http.Request) {
	dbpool := routeHelper.GetDB(r)
	logger := routeHelper.GetLogger(r)

	pager, err := dbpool.FindAllPosts(&db.Pager{Num: 50, Page: 0})
	if err != nil {
		logger.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ts, err := template.ParseFiles("./html/rss.page.tmpl", "./html/list.partial.tmpl")
	if err != nil {
		logger.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	feed := &feeds.Feed{
		Title:       "lists.sh discovery feed",
		Link:        &feeds.Link{Href: "https://lists.sh/read"},
		Description: "lists.sh latest posts",
		Author:      &feeds.Author{Name: "lists.sh"},
		Created:     time.Now(),
	}

	var feedItems []*feeds.Item
	for _, post := range pager.Data {
		parsed := pkg.ParseText(post.Text)
		var tpl bytes.Buffer
		data := &PostPageData{
			ListType: parsed.MetaData.ListType,
			Items:    parsed.Items,
		}
		if err := ts.Execute(&tpl, data); err != nil {
			continue
		}
		feedItems = append(feedItems, &feeds.Item{
			Id:          post.ID,
			Title:       post.Title,
			Link:        &feeds.Link{Href: postURL(post)},
			Description: post.Description,
			Content:     tpl.String(),
			Created:     *post.PublishAt,
		})
	}
	feed.Items = feedItems

	rss, err := feed.ToAtom()
	if err != nil {
		logger.Fatal(err)
		http.Error(w, "Could not generate atom rss feed", http.StatusInternalServerError)
	}

	w.Header().Add("Content-Type", "application/atom+xml")
	fmt.Fprintf(w, rss)
}

func serveFile(file string, contentType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := routeHelper.GetLogger(r)

		contents, err := ioutil.ReadFile(fmt.Sprintf("./public/%s", file))
		if err != nil {
			logger.Error(err)
			http.Error(w, "file not found", 404)
		}
		w.Header().Add("Content-Type", contentType)
		w.Write(contents)
	}
}

var routes = []routeHelper.Route{
	routeHelper.NewRoute("GET", "/", createPageHandler("./html/marketing.page.tmpl")),
	routeHelper.NewRoute("GET", "/spec", createPageHandler("./html/spec.page.tmpl")),
	routeHelper.NewRoute("GET", "/ops", createPageHandler("./html/ops.page.tmpl")),
	routeHelper.NewRoute("GET", "/privacy", createPageHandler("./html/privacy.page.tmpl")),
	routeHelper.NewRoute("GET", "/help", createPageHandler("./html/help.page.tmpl")),
	routeHelper.NewRoute("GET", "/main.css", serveFile("main.css", "text/css")),
	routeHelper.NewRoute("GET", "/card.png", serveFile("card.png", "image/png")),
	routeHelper.NewRoute("GET", "/favicon-16x16.png", serveFile("favicon-16x16.png", "image/png")),
	routeHelper.NewRoute("GET", "/favicon-32x32.png", serveFile("favicon-32x32.png", "image/png")),
	routeHelper.NewRoute("GET", "/apple-touch-icon.png", serveFile("apple-touch-icon.png", "image/png")),
	routeHelper.NewRoute("GET", "/favicon.ico", serveFile("favicon.ico", "image/x-icon")),
	routeHelper.NewRoute("GET", "/robots.txt", serveFile("robots.txt", "text/plain")),
	routeHelper.NewRoute("GET", "/transparency", transparencyHandler),
	routeHelper.NewRoute("GET", "/read", readHandler),
	routeHelper.NewRoute("GET", "/rss", rssHandler),
	routeHelper.NewRoute("GET", "/rss.xml", rssHandler),
	routeHelper.NewRoute("GET", "/atom.xml", rssHandler),
	routeHelper.NewRoute("GET", "/feed.xml", rssHandler),
	routeHelper.NewRoute("GET", "/([^/]+)", blogHandler),
	routeHelper.NewRoute("GET", "/([^/]+)/rss", rssBlogHandler),
	routeHelper.NewRoute("GET", "/([^/]+)/([^/]+)", postHandler),
}

func StartServer() {
	db := postgres.NewDB()
	defer db.Close()
	logger := internal.CreateLogger()

	handler := routeHelper.CreateServe(routes, db, logger)
	router := http.HandlerFunc(handler)

	port := internal.GetEnv("LISTS_WEB_PORT", "3000")
	portStr := fmt.Sprintf(":%s", port)
	logger.Infof("Starting server on port %s", port)
	logger.Fatal(http.ListenAndServe(portStr, router))
}
