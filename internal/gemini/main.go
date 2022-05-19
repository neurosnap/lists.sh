package gemini

import (
	"bytes"
	"context"
	"fmt"
	html "html/template"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"text/template"
	"time"

	"git.sr.ht/~adnano/go-gemini"
	"git.sr.ht/~adnano/go-gemini/certificate"
	feeds "git.sr.ht/~aw/gorilla-feeds"
	"github.com/neurosnap/lists.sh/internal"
	"github.com/neurosnap/lists.sh/internal/api"
	"github.com/neurosnap/lists.sh/internal/db"
	"github.com/neurosnap/lists.sh/internal/db/postgres"
	"github.com/neurosnap/lists.sh/pkg"
	"go.uber.org/zap"
)

type ctxKey struct{}
type ctxDBKey struct{}
type ctxLoggerKey struct{}
type ctxSubdomainKey struct{}

func GetLogger(ctx context.Context) *zap.SugaredLogger {
	return ctx.Value(ctxLoggerKey{}).(*zap.SugaredLogger)
}

func GetDB(ctx context.Context) db.DB {
	return ctx.Value(ctxDBKey{}).(db.DB)
}

func GetField(ctx context.Context, index int) string {
	fields := ctx.Value(ctxKey{}).([]string)
	return fields[index]
}

type Route struct {
	regex   *regexp.Regexp
	handler gemini.HandlerFunc
}

func NewRoute(pattern string, handler gemini.HandlerFunc) Route {
	return Route{
		regexp.MustCompile("^" + pattern + "$"),
		handler,
	}
}

type ServeFn func(context.Context, gemini.ResponseWriter, *gemini.Request)

func CreateServe(routes []Route, dbpool db.DB, logger *zap.SugaredLogger) ServeFn {
	return func(ctx context.Context, w gemini.ResponseWriter, r *gemini.Request) {
		curRoutes := routes

		for _, route := range curRoutes {
			matches := route.regex.FindStringSubmatch(r.URL.Path)
			if len(matches) > 0 {
				ctx = context.WithValue(ctx, ctxLoggerKey{}, logger)
				ctx = context.WithValue(ctx, ctxDBKey{}, dbpool)
				ctx = context.WithValue(ctx, ctxKey{}, matches[1:])
				route.handler(ctx, w, r)
				return
			}
		}
		w.WriteHeader(gemini.StatusTemporaryFailure, "Internal Service Error")
	}
}

func renderTemplate(templates []string) (*template.Template, error) {
	files := make([]string, len(templates))
	copy(files, templates)
	files = append(
		files,
		"./gmi/footer.partial.tmpl",
		"./gmi/marketing-footer.partial.tmpl",
		"./gmi/base.layout.tmpl",
	)

	ts, err := template.ParseFiles(files...)
	if err != nil {
		return nil, err
	}
	return ts, nil
}

func createPageHandler(fname string) gemini.HandlerFunc {
	return func(ctx context.Context, w gemini.ResponseWriter, r *gemini.Request) {
		logger := GetLogger(ctx)
		ts, err := renderTemplate([]string{fname})

		if err != nil {
			logger.Error(err)
			w.WriteHeader(gemini.StatusTemporaryFailure, "Internal Service Error")
			return
		}

		data := api.PageData{
			Site: internal.SiteData,
		}
		err = ts.Execute(w, data)
		if err != nil {
			logger.Error(err)
			w.WriteHeader(gemini.StatusTemporaryFailure, "Internal Service Error")
		}
	}
}

func blogHandler(ctx context.Context, w gemini.ResponseWriter, r *gemini.Request) {
	username := GetField(ctx, 0)
	dbpool := GetDB(ctx)
	logger := GetLogger(ctx)

	user, err := dbpool.UserForName(username)
	if err != nil {
		logger.Infof("blog not found: %s", username)
		w.WriteHeader(gemini.StatusNotFound, "blog not found")
		return
	}
	posts, err := dbpool.PostsForUser(user.ID)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(gemini.StatusTemporaryFailure, "could not fetch posts for blog")
		return
	}

	ts, err := renderTemplate([]string{
		"./gmi/blog.page.tmpl",
		"./gmi/list.partial.tmpl",
	})

	if err != nil {
		logger.Error(err)
		w.WriteHeader(gemini.StatusTemporaryFailure, err.Error())
		return
	}

	headerTxt := &api.HeaderTxt{
		Title: api.GetBlogName(username),
		Bio:   "",
	}
	readmeTxt := &api.ReadmeTxt{}

	postCollection := make([]api.PostItemData, 0, len(posts))
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
			p := api.PostItemData{
				URL:          html.URL(internal.PostURL(post.Username, post.Filename)),
				BlogURL:      html.URL(internal.BlogURL(post.Username)),
				Title:        internal.FilenameToTitle(post.Filename, post.Title),
				PublishAt:    post.PublishAt.Format("02 Jan, 2006"),
				PublishAtISO: post.PublishAt.Format(time.RFC3339),
			}
			postCollection = append(postCollection, p)
		}
	}

	data := api.BlogPageData{
		Site:      internal.SiteData,
		PageTitle: headerTxt.Title,
		URL:       html.URL(internal.BlogURL(username)),
		RSSURL:    html.URL(internal.RssBlogURL(username)),
		Readme:    readmeTxt,
		Header:    headerTxt,
		Username:  username,
		Posts:     postCollection,
	}

	err = ts.Execute(w, data)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(gemini.StatusTemporaryFailure, err.Error())
	}
}

func readHandler(ctx context.Context, w gemini.ResponseWriter, r *gemini.Request) {
	dbpool := GetDB(ctx)
	logger := GetLogger(ctx)

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pager, err := dbpool.FindAllPosts(&db.Pager{Num: 20, Page: page})
	if err != nil {
		logger.Error(err)
		w.WriteHeader(gemini.StatusTemporaryFailure, err.Error())
		return
	}

	ts, err := renderTemplate([]string{
		"./gmi/read.page.tmpl",
	})

	if err != nil {
		w.WriteHeader(gemini.StatusTemporaryFailure, err.Error())
		return
	}

	nextPage := ""
	if page < pager.Total-1 {
		nextPage = fmt.Sprintf("/read?page=%d", page+1)
	}

	prevPage := ""
	if page > 0 {
		prevPage = fmt.Sprintf("/read?page=%d", page-1)
	}

	data := api.ReadPageData{
		Site:     internal.SiteData,
		NextPage: nextPage,
		PrevPage: prevPage,
	}
	for _, post := range pager.Data {
		item := api.PostItemData{
			URL:          html.URL(internal.PostURL(post.Username, post.Filename)),
			BlogURL:      html.URL(internal.BlogURL(post.Username)),
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
		w.WriteHeader(gemini.StatusTemporaryFailure, err.Error())
	}
}

func postHandler(ctx context.Context, w gemini.ResponseWriter, r *gemini.Request) {
	username := GetField(ctx, 0)
	filename := GetField(ctx, 1)

	dbpool := GetDB(ctx)
	logger := GetLogger(ctx)

	user, err := dbpool.UserForName(username)
	if err != nil {
		logger.Infof("blog not found: %s", username)
		w.WriteHeader(gemini.StatusNotFound, "blog not found")
		return
	}

	header, _ := dbpool.FindPostWithFilename("_header", user.ID)
	blogName := api.GetBlogName(username)
	if header != nil {
		headerParsed := pkg.ParseText(header.Text)
		if headerParsed.MetaData.Title != "" {
			blogName = headerParsed.MetaData.Title
		}
	}

	post, err := dbpool.FindPostWithFilename(filename, user.ID)
	if err != nil {
		logger.Infof("post not found %s/%s", username, filename)
		w.WriteHeader(gemini.StatusNotFound, "post not found")
		return
	}

	parsedText := pkg.ParseText(post.Text)

	data := api.PostPageData{
		Site:         internal.SiteData,
		PageTitle:    api.GetPostTitle(post),
		URL:          html.URL(internal.PostURL(post.Username, post.Filename)),
		BlogURL:      html.URL(internal.BlogURL(username)),
		Description:  post.Description,
		ListType:     parsedText.MetaData.ListType,
		Title:        internal.FilenameToTitle(post.Filename, post.Title),
		PublishAt:    post.PublishAt.Format("02 Jan, 2006"),
		PublishAtISO: post.PublishAt.Format(time.RFC3339),
		Username:     username,
		BlogName:     blogName,
		Items:        parsedText.Items,
	}

	ts, err := renderTemplate([]string{
		"./gmi/post.page.tmpl",
		"./gmi/list.partial.tmpl",
	})

	if err != nil {
		w.WriteHeader(gemini.StatusTemporaryFailure, err.Error())
		return
	}

	err = ts.Execute(w, data)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(gemini.StatusTemporaryFailure, err.Error())
	}
}

func transparencyHandler(ctx context.Context, w gemini.ResponseWriter, r *gemini.Request) {
	dbpool := GetDB(ctx)
	logger := GetLogger(ctx)

	analytics, err := dbpool.SiteAnalytics()
	if err != nil {
		logger.Error(err)
		w.WriteHeader(gemini.StatusTemporaryFailure, err.Error())
		return
	}

	ts, err := template.ParseFiles(
		"./gmi/transparency.page.tmpl",
		"./gmi/footer.partial.tmpl",
		"./gmi/marketing-footer.partial.tmpl",
		"./gmi/base.layout.tmpl",
	)

	if err != nil {
		w.WriteHeader(gemini.StatusTemporaryFailure, err.Error())
		return
	}

	data := api.TransparencyPageData{
		Site:      internal.SiteData,
		Analytics: analytics,
	}
	err = ts.Execute(w, data)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(gemini.StatusTemporaryFailure, err.Error())
	}
}

func rssBlogHandler(ctx context.Context, w gemini.ResponseWriter, r *gemini.Request) {
	username := GetField(ctx, 0)
	dbpool := GetDB(ctx)
	logger := GetLogger(ctx)

	user, err := dbpool.UserForName(username)
	if err != nil {
		logger.Infof("rss feed not found: %s", username)
		w.WriteHeader(gemini.StatusNotFound, "rss feed not found")
		return
	}
	posts, err := dbpool.PostsForUser(user.ID)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(gemini.StatusTemporaryFailure, err.Error())
		return
	}

	ts, err := template.ParseFiles("./gmi/rss.page.tmpl", "./gmi/list.partial.tmpl")
	if err != nil {
		logger.Error(err)
		w.WriteHeader(gemini.StatusTemporaryFailure, err.Error())
		return
	}

	headerTxt := &api.HeaderTxt{
		Title: api.GetBlogName(username),
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
		Link:        &feeds.Link{Href: internal.BlogURL(username)},
		Description: headerTxt.Bio,
		Author:      &feeds.Author{Name: username},
		Created:     time.Now(),
	}

	var feedItems []*feeds.Item
	for _, post := range posts {
		parsed := pkg.ParseText(post.Text)
		var tpl bytes.Buffer
		data := &api.PostPageData{
			ListType: parsed.MetaData.ListType,
			Items:    parsed.Items,
		}
		if err := ts.Execute(&tpl, data); err != nil {
			continue
		}
		feedItems = append(feedItems, &feeds.Item{
			Id:          post.ID,
			Title:       post.Title,
			Link:        &feeds.Link{Href: internal.PostURL(post.Username, post.Filename)},
			Description: post.Description,
			Content:     tpl.String(),
			Created:     *post.PublishAt,
		})
	}
	feed.Items = feedItems

	rss, err := feed.ToAtom()
	if err != nil {
		logger.Fatal(err)
		w.WriteHeader(gemini.StatusTemporaryFailure, "Could not generate atom rss feed")
		return
	}

	// w.Header().Add("Content-Type", "application/atom+xml")
	fmt.Fprintf(w, rss)
}

func rssHandler(ctx context.Context, w gemini.ResponseWriter, r *gemini.Request) {
	dbpool := GetDB(ctx)
	logger := GetLogger(ctx)

	pager, err := dbpool.FindAllPosts(&db.Pager{Num: 50, Page: 0})
	if err != nil {
		logger.Error(err)
		w.WriteHeader(gemini.StatusTemporaryFailure, err.Error())
		return
	}

	ts, err := template.ParseFiles("./gmi/rss.page.tmpl", "./gmi/list.partial.tmpl")
	if err != nil {
		logger.Error(err)
		w.WriteHeader(gemini.StatusTemporaryFailure, err.Error())
		return
	}

	feed := &feeds.Feed{
		Title:       fmt.Sprintf("%s discovery feed", internal.Domain),
		Link:        &feeds.Link{Href: internal.ReadURL()},
		Description: fmt.Sprintf("%s latest posts", internal.Domain),
		Author:      &feeds.Author{Name: internal.Domain},
		Created:     time.Now(),
	}

	var feedItems []*feeds.Item
	for _, post := range pager.Data {
		parsed := pkg.ParseText(post.Text)
		var tpl bytes.Buffer
		data := &api.PostPageData{
			ListType: parsed.MetaData.ListType,
			Items:    parsed.Items,
		}
		if err := ts.Execute(&tpl, data); err != nil {
			continue
		}
		feedItems = append(feedItems, &feeds.Item{
			Id:          post.ID,
			Title:       post.Title,
			Link:        &feeds.Link{Href: internal.PostURL(post.Username, post.Filename)},
			Description: post.Description,
			Content:     tpl.String(),
			Created:     *post.PublishAt,
		})
	}
	feed.Items = feedItems

	rss, err := feed.ToAtom()
	if err != nil {
		logger.Fatal(err)
		w.WriteHeader(gemini.StatusTemporaryFailure, "Could not generate atom rss feed")
	}

	// w.Header().Add("Content-Type", "application/atom+xml")
	fmt.Fprintf(w, rss)
}

func StartServer() {
	db := postgres.NewDB()
	logger := internal.CreateLogger()

	certificates := &certificate.Store{}
	certificates.Register("localhost")
	certificates.Register("lists.sh")
	certificates.Register("*.lists.sh")
	if err := certificates.Load("/var/lib/gemini/certs"); err != nil {
		logger.Fatal(err)
	}

	routes := []Route{
		NewRoute("/", createPageHandler("./gmi/marketing.page.tmpl")),
		NewRoute("/spec", createPageHandler("./gmi/spec.page.tmpl")),
		NewRoute("/help", createPageHandler("./gmi/help.page.tmpl")),
		NewRoute("/ops", createPageHandler("./gmi/ops.page.tmpl")),
		NewRoute("/privacy", createPageHandler("./gmi/privacy.page.tmpl")),
		NewRoute("/transparency", transparencyHandler),
		NewRoute("/read", readHandler),
		NewRoute("/rss", rssHandler),
		NewRoute("/([^/]+)", blogHandler),
		NewRoute("/([^/]+)/rss", rssBlogHandler),
		NewRoute("/([^/]+)/([^/]+)", postHandler),
	}
	handler := CreateServe(routes, db, logger)
	router := gemini.HandlerFunc(handler)

	server := &gemini.Server{
		Addr:           "0.0.0.0:1965",
		Handler:        gemini.LoggingMiddleware(router),
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   1 * time.Minute,
		GetCertificate: certificates.Get,
	}

	// Listen for interrupt signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	errch := make(chan error)
	go func() {
		logger.Info("Starting server")
		ctx := context.Background()
		errch <- server.ListenAndServe(ctx)
	}()

	select {
	case err := <-errch:
		logger.Fatal(err)
	case <-c:
		// Shutdown the server
		logger.Info("Shutting down...")
		db.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		err := server.Shutdown(ctx)
		if err != nil {
			logger.Fatal(err)
		}
	}
}
