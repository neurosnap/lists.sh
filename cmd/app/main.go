package main

// An example Bubble Tea server. This will put an ssh session into alt screen
// and continually print up to date terminal information.

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/wish"
	bm "github.com/charmbracelet/wish/bubbletea"
	lm "github.com/charmbracelet/wish/logging"
	"github.com/gliderlabs/ssh"
	"github.com/muesli/reflow/indent"
	"github.com/muesli/reflow/wordwrap"
	"github.com/muesli/reflow/wrap"
	"github.com/neurosnap/lists.sh/internal"
	"github.com/neurosnap/lists.sh/internal/db"
	"github.com/neurosnap/lists.sh/internal/db/postgres"
	"github.com/neurosnap/lists.sh/internal/ui/common"
	"github.com/neurosnap/lists.sh/internal/ui/info"
	"github.com/neurosnap/lists.sh/internal/ui/posts"
	"github.com/neurosnap/lists.sh/internal/ui/username"
)

const host = "localhost"
const port = 23234

// status is used to indicate a high level application state.
type status int

const (
	statusInit status = iota
	statusReady
	statusLinking
	statusBrowsingPosts
	statusSettingUsername
	statusQuitting
	statusError
)

func (s status) String() string {
	return [...]string{
		"initializing",
		"ready",
		"setting username",
		"browsing posts",
		"quitting",
		"error",
	}[s]
}

// menuChoice represents a chosen menu item.
type menuChoice int

// menu choices
const (
	setUserChoice menuChoice = iota
	postsChoice
	exitChoice
	unsetChoice // set when no choice has been made
)

// menu text corresponding to menu choices. these are presented to the user.
var menuChoices = map[menuChoice]string{
	setUserChoice: "Set username",
	postsChoice:   "Manage posts",
	exitChoice:    "Exit",
}

type SSHServer struct{}

func (me *SSHServer) authHandler(ctx ssh.Context, key ssh.PublicKey) bool {
	return true
}

func main() {
	sshServer := &SSHServer{}
	s, err := wish.NewServer(
		wish.WithAddress(fmt.Sprintf("%s:%d", host, port)),
		wish.WithHostKeyPath(".ssh/term_info_ed25519"),
		wish.WithPublicKeyAuth(sshServer.authHandler),
		wish.WithMiddleware(
			bm.Middleware(teaHandler),
			lm.Middleware(),
		),
	)
	if err != nil {
		log.Fatalln(err)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	log.Printf("Starting SSH server on %s:%d", host, port)
	go func() {
		if err = s.ListenAndServe(); err != nil {
			log.Fatalln(err)
		}
	}()

	<-done
	log.Println("Stopping SSH server")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer func() { cancel() }()
	if err := s.Shutdown(ctx); err != nil {
		log.Fatalln(err)
	}
}

var (
	spinnerStyle = lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#8E8E8E", Dark: "#747373"})
)

// NewSpinner returns a spinner model.
func NewSpinner() spinner.Model {
	s := spinner.NewModel()
	s.Spinner = spinner.Dot
	s.Style = spinnerStyle
	return s
}

type GotDBMsg db.DB

// You can wire any Bubble Tea model up to the middleware with a function that
// handles the incoming ssh.Session. Here we just grab the terminal info and
// pass it to the new model. You can also return tea.ProgramOptions (such as
// teaw.WithAltScreen) on a session by session basis
func teaHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
	_, _, active := s.Pty()
	if !active {
		fmt.Println("no active terminal, skipping")
		return nil, nil
	}
	key, err := internal.KeyText(s)
	if err != nil {
		log.Println(err)
	}

	databaseUrl := os.Getenv("DATABASE_URL")
	dbpool := postgres.NewDB(databaseUrl)
	user, err := FindOrRegisterUser(dbpool, key)
	if err != nil {
		fmt.Println(err)
	}

	m := model{
		publicKey:  key,
		dbpool:     dbpool,
		user:       user,
		status:     statusInit,
		menuChoice: unsetChoice,
		spinner:    common.NewSpinner(),
	}

	return m, []tea.ProgramOption{tea.WithAltScreen()}
}

// Just a generic tea.Model to demo terminal information of ssh.
type model struct {
	publicKey     string
	dbpool        db.DB
	user          *db.User
	err           error
	status        status
	menuIndex     int
	menuChoice    menuChoice
	terminalWidth int
	styles        common.Styles
	info          info.Model
	spinner       spinner.Model
	username      username.Model
	posts         posts.Model
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		spinner.Tick,
	)
}

func FindOrRegisterUser(dbpool db.DB, publicKey string) (*db.User, error) {
	user, err := dbpool.UserForKey(publicKey)
	if user != nil {
		return user, nil
	}

	userID, err := dbpool.AddUser()
	if err != nil {
		return nil, err
	}

	err = dbpool.LinkUserKey(userID, publicKey)
	if err != nil {
		return nil, err
	}

	user, err = dbpool.UserForKey(publicKey)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmds []tea.Cmd
		cmd  tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.terminalWidth = msg.Width
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}

		if m.status == statusReady { // Process keys for the menu
			switch msg.String() {
			// Quit
			case "q", "esc":
				m.status = statusQuitting
				return m, tea.Quit

			// Prev menu item
			case "up", "k":
				m.menuIndex--
				if m.menuIndex < 0 {
					m.menuIndex = len(menuChoices) - 1
				}

			// Select menu item
			case "enter":
				m.menuChoice = menuChoice(m.menuIndex)

			// Next menu item
			case "down", "j":
				m.menuIndex++
				if m.menuIndex >= len(menuChoices) {
					m.menuIndex = 0
				}
			}
		}
	case username.NameSetMsg:
		m.status = statusReady
		m.info.User.Name = string(msg)
		m.user = m.info.User
		m.username = username.NewModel(m.dbpool, m.user) // reset the state
	}

	switch m.status {
	case statusInit:
		m.username = username.NewModel(m.dbpool, m.user)
		m.info = info.NewModel(m.user)
		m.posts = posts.NewModel(m.dbpool, m.user)
		m.status = statusReady
	}

	m, cmd = updateChilden(msg, m)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func updateChilden(msg tea.Msg, m model) (model, tea.Cmd) {
	var cmd tea.Cmd

	switch m.status {
	// User info
	case statusBrowsingPosts:
		newModel, newCmd := m.posts.Update(msg)
		postsModel, ok := newModel.(posts.Model)
		if !ok {
			panic("could not perform assertion on posts model")
		}
		m.posts = postsModel
		cmd = newCmd

		if m.posts.Exit {
			m.posts = posts.NewModel(m.dbpool, m.user)
			m.status = statusReady
		} else if m.posts.Quit {
			m.status = statusQuitting
			return m, tea.Quit
		}
	// Username tool
	case statusSettingUsername:
		m.username, cmd = username.Update(msg, m.username)
		if m.username.Done {
			m.username = username.NewModel(m.dbpool, m.user) // reset the state
			m.status = statusReady
		} else if m.username.Quit {
			m.status = statusQuitting
			return m, tea.Quit
		}
	}

	// Handle the menu
	switch m.menuChoice {
	case setUserChoice:
		m.status = statusSettingUsername
		m.menuChoice = unsetChoice
		cmd = username.InitialCmd()
	case postsChoice:
		m.status = statusBrowsingPosts
		m.menuChoice = unsetChoice
		cmd = posts.LoadPosts(m.posts)
	case exitChoice:
		m.status = statusQuitting
		cmd = tea.Quit
	}

	return m, cmd
}

func (m model) menuView() string {
	var s string
	for i := 0; i < len(menuChoices); i++ {
		e := "  "
		menuItem := menuChoices[menuChoice(i)]
		if i == m.menuIndex {
			e = m.styles.SelectionMarker.String() +
				m.styles.SelectedMenuItem.Render(menuItem)
		} else {
			e += menuItem
		}
		if i < len(menuChoices)-1 {
			e += "\n"
		}
		s += e
	}

	return s
}

func (m model) quitView() string {
	if m.err != nil {
		return fmt.Sprintf("Uh oh, there’s been an error: %s\n", m.err)
	}
	return "Thanks for using lists.sh!\n"
}

func footerView(m model) string {
	if m.err != nil {
		return m.errorView(m.err)
	}
	return "\n\n" + common.HelpView("j/k, ↑/↓: choose", "enter: select")
}

func (m model) errorView(err error) string {
	head := m.styles.Error.Render("Error: ")
	body := m.styles.Subtle.Render(err.Error())
	msg := m.styles.Wrap.Render(head + body)
	return "\n\n" + indent.String(msg, 2)
}

func (m model) View() string {
	w := m.terminalWidth - m.styles.App.GetHorizontalFrameSize()
	s := m.styles.Logo.String() + "\n\n"
	switch m.status {
	case statusReady:
		s += m.info.View()
		s += "\n\n" + m.menuView()
		s += footerView(m)
	case statusSettingUsername:
		s += username.View(m.username)
	case statusBrowsingPosts:
		s += m.posts.View()
	}
	return m.styles.App.Render(wrap.String(wordwrap.String(s, w), w))
}
