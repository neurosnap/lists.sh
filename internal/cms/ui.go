package cms

// An example Bubble Tea server. This will put an ssh session into alt screen
// and continually print up to date terminal information.

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gliderlabs/ssh"
	"github.com/muesli/reflow/indent"
	"github.com/muesli/reflow/wordwrap"
	"github.com/muesli/reflow/wrap"
	"github.com/neurosnap/lists.sh/internal"
	"github.com/neurosnap/lists.sh/internal/db"
	"github.com/neurosnap/lists.sh/internal/db/postgres"
	"github.com/neurosnap/lists.sh/internal/ui/account"
	"github.com/neurosnap/lists.sh/internal/ui/common"
	"github.com/neurosnap/lists.sh/internal/ui/info"
	"github.com/neurosnap/lists.sh/internal/ui/posts"
	"github.com/neurosnap/lists.sh/internal/ui/username"
)

// status is used to indicate a high level application state.
type status int

const (
	statusInit status = iota
	statusReady
	statusNoAccount
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
func Handler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
	logger := internal.CreateLogger()

	_, _, active := s.Pty()
	if !active {
		logger.Error("no active terminal, skipping")
		return nil, nil
	}
	key, err := internal.KeyText(s)
	if err != nil {
		logger.Error(err)
	}

	sshUser := s.User()

	dbpool := postgres.NewDB()
	user := FindUser(dbpool, key, sshUser)

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
	createAccount account.CreateModel
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		spinner.Tick,
	)
}

func FindUser(dbpool db.DB, publicKey string, sshUser string) *db.User {
	logger := internal.CreateLogger()
	var user *db.User
	if sshUser != "" {
		logger.Infof("Finding user based on ssh user (%s)", sshUser)
		user, _ = dbpool.UserForName(sshUser)
	} else {
		logger.Infof("Finding user based on public key (%s)", publicKey)
		user, _ = dbpool.UserForKey(publicKey)
	}
	return user
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
			m.dbpool.Close()
			return m, tea.Quit
		}

		if m.status == statusReady { // Process keys for the menu
			switch msg.String() {
			// Quit
			case "q", "esc":
				m.status = statusQuitting
				m.dbpool.Close()
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
	case account.CreateAccountMsg:
		m.status = statusReady
		m.info.User = msg
		m.user = msg
		m.createAccount = account.NewCreateModel(m.dbpool, m.publicKey)
	}

	switch m.status {
	case statusInit:
		m.username = username.NewModel(m.dbpool, m.user)
		m.info = info.NewModel(m.user)
		m.posts = posts.NewModel(m.dbpool, m.user)
		m.createAccount = account.NewCreateModel(m.dbpool, m.publicKey)
		if m.user == nil {
			m.status = statusNoAccount
		} else {
			m.status = statusReady
		}
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
	case statusNoAccount:
		m.createAccount, cmd = account.Update(msg, m.createAccount)
		if m.createAccount.Done {
			m.createAccount = account.NewCreateModel(m.dbpool, m.publicKey) // reset the state
			m.status = statusReady
		} else if m.createAccount.Quit {
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
		m.dbpool.Close()
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
	case statusNoAccount:
		s += account.View(m.createAccount)
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
