package info

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/neurosnap/lists.sh/internal"
	"github.com/neurosnap/lists.sh/internal/db"
	"github.com/neurosnap/lists.sh/internal/ui/common"
)

type errMsg struct {
	err error
}

// Error satisfies the error interface.
func (e errMsg) Error() string {
	return e.err.Error()
}

// Model stores the state of the info user interface.
type Model struct {
	Quit   bool // signals it's time to exit the whole application
	Err    error
	User   *db.User
	styles common.Styles
}

// NewModel returns a new Model in its initial state.
func NewModel(user *db.User) Model {
	return Model{
		Quit:   false,
		User:   user,
		styles: common.DefaultStyles(),
	}
}

// Update is the Bubble Tea update loop.
func Update(msg tea.Msg, m Model) (Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q":
			m.Quit = true
			return m, nil
		}
	case errMsg:
		// If there's an error we print the error and exit
		m.Err = msg
		m.Quit = true
		return m, nil
	}

	return m, cmd
}

// View renders the current view from the model.
func (m Model) View() string {
	if m.Err != nil {
		return "error: " + m.Err.Error()
	} else if m.User == nil {
		return " Authenticating..."
	}
	return m.bioView()
}

func (m Model) bioView() string {
	var username string
	if m.User.Name != "" {
		username = m.User.Name
	} else {
		username = m.styles.Subtle.Render("(none set)")
	}
	return common.KeyValueView(
		"Username", username,
		"Blog URL", internal.BlogURL(username),
		"Public key", m.User.PublicKey.Key,
		"Joined", m.User.CreatedAt.Format("02 Jan 2006"),
	)
}
