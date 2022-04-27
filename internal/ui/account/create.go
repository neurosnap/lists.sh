package account

import (
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	input "github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/neurosnap/lists.sh/internal/db"
	"github.com/neurosnap/lists.sh/internal/ui/common"
)

type state int

const (
	ready state = iota
	submitting
)

// index specifies the UI element that's in focus.
type index int

const (
	textInput index = iota
	okButton
	cancelButton
)

type CreateAccountMsg *db.User

// NameTakenMsg is sent when the requested username has already been taken.
type NameTakenMsg struct{}

// NameInvalidMsg is sent when the requested username has failed validation.
type NameInvalidMsg struct{}

type errMsg struct{ err error }

func (e errMsg) Error() string { return e.err.Error() }

// Model holds the state of the username UI.
type CreateModel struct {
	Done bool // true when it's time to exit this view
	Quit bool // true when the user wants to quit the whole program

	dbpool    db.DB
	publicKey string
	styles    common.Styles
	state     state
	newName   string
	index     index
	errMsg    string
	input     input.Model
	spinner   spinner.Model
}

// updateFocus updates the focused states in the model based on the current
// focus index.
func (m *CreateModel) updateFocus() {
	if m.index == textInput && !m.input.Focused() {
		m.input.Focus()
		m.input.Prompt = m.styles.FocusedPrompt.String()
	} else if m.index != textInput && m.input.Focused() {
		m.input.Blur()
		m.input.Prompt = m.styles.Prompt.String()
	}
}

// Move the focus index one unit forward.
func (m *CreateModel) indexForward() {
	m.index++
	if m.index > cancelButton {
		m.index = textInput
	}

	m.updateFocus()
}

// Move the focus index one unit backwards.
func (m *CreateModel) indexBackward() {
	m.index--
	if m.index < textInput {
		m.index = cancelButton
	}

	m.updateFocus()
}

// NewModel returns a new username model in its initial state.
func NewCreateModel(dbpool db.DB, publicKey string) CreateModel {
	st := common.DefaultStyles()

	im := input.NewModel()
	im.CursorStyle = st.Cursor
	im.Placeholder = "erock"
	im.Prompt = st.FocusedPrompt.String()
	im.CharLimit = 50
	im.Focus()

	return CreateModel{
		Done:      false,
		Quit:      false,
		dbpool:    dbpool,
		styles:    st,
		state:     ready,
		newName:   "",
		index:     textInput,
		errMsg:    "",
		input:     im,
		spinner:   common.NewSpinner(),
		publicKey: publicKey,
	}
}

// Init is the Bubble Tea initialization function.
func Init(dbpool db.DB, publicKey string) func() (CreateModel, tea.Cmd) {
	return func() (CreateModel, tea.Cmd) {
		m := NewCreateModel(dbpool, publicKey)
		return m, InitialCmd()
	}
}

// InitialCmd returns the initial command.
func InitialCmd() tea.Cmd {
	return input.Blink
}

// Update is the Bubble Tea update loop.
func Update(msg tea.Msg, m CreateModel) (CreateModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC: // quit
			m.Quit = true
			return m, nil
		case tea.KeyEscape: // exit this mini-app
			m.Done = true
			return m, nil

		default:
			// Ignore keys if we're submitting
			if m.state == submitting {
				return m, nil
			}

			switch msg.String() {
			case "tab":
				m.indexForward()
			case "shift+tab":
				m.indexBackward()
			case "l", "k", "right":
				if m.index != textInput {
					m.indexForward()
				}
			case "h", "j", "left":
				if m.index != textInput {
					m.indexBackward()
				}
			case "up", "down":
				if m.index == textInput {
					m.indexForward()
				} else {
					m.index = textInput
					m.updateFocus()
				}
			case "enter":
				switch m.index {
				case textInput:
					fallthrough
				case okButton: // Submit the form
					m.state = submitting
					m.errMsg = ""
					m.newName = strings.TrimSpace(m.input.Value())

					return m, tea.Batch(
						createAccount(m), // fire off the command, too
						spinner.Tick,
					)
				case cancelButton: // Exit
					m.Quit = true
					return m, nil
				}
			}

			// Pass messages through to the input element if that's the element
			// in focus
			if m.index == textInput {
				var cmd tea.Cmd
				m.input, cmd = m.input.Update(msg)

				return m, cmd
			}

			return m, nil
		}

	case NameTakenMsg:
		m.state = ready
		m.errMsg = m.styles.Subtle.Render("Sorry, ") +
			m.styles.Error.Render(m.newName) +
			m.styles.Subtle.Render(" is taken.")

		return m, nil

	case NameInvalidMsg:
		m.state = ready
		head := m.styles.Error.Render("Invalid name. ")
		body := m.styles.Subtle.Render("Names can only contain plain letters and numbers and must be less than 50 characters. And no emojis, kiddo.")
		m.errMsg = m.styles.Wrap.Render(head + body)

		return m, nil

	case errMsg:
		m.state = ready
		head := m.styles.Error.Render("Oh, what? There was a curious error we were not expecting. ")
		body := m.styles.Subtle.Render(msg.Error())
		m.errMsg = m.styles.Wrap.Render(head + body)

		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)

		return m, cmd

	default:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg) // Do we still need this?

		return m, cmd
	}
}

// View renders current view from the model.
func View(m CreateModel) string {
	s := "lists.sh\n"
	s += "A microblog for lists\n\n"
	s += "To get started, enter a username.\n"
	s += "Then create a folder locally (e.g. ~/blog).\n"
	s += "Then write your lists in plain text files (e.g. hello-world.txt).\n"
	s += "Finally, send your list files to us:\n\n"
	s += "scp ~/blog/*.txt lists.sh:/\n\n"
	s += "Enter a username\n\n"
	s += m.input.View() + "\n\n"

	if m.state == submitting {
		s += spinnerView(m)
	} else {
		s += common.OKButtonView(m.index == 1, true)
		s += " " + common.CancelButtonView(m.index == 2, false)
		if m.errMsg != "" {
			s += "\n\n" + m.errMsg
		}
	}

	return s
}

func nameSetView(m CreateModel) string {
	return "OK! Your username is " + m.newName
}

func spinnerView(m CreateModel) string {
	return m.spinner.View() + " Creating account..."
}

func registerUser(m CreateModel) (*db.User, error) {
	userID, err := m.dbpool.AddUser()
	if err != nil {
		return nil, err
	}

	err = m.dbpool.LinkUserKey(userID, m.publicKey)
	if err != nil {
		return nil, err
	}

	user, err := m.dbpool.UserForKey(m.publicKey)
	if err != nil {
		return nil, err
	}

	return user, nil

}

// Attempt to update the username on the server.
func createAccount(m CreateModel) tea.Cmd {
	return func() tea.Msg {
		if m.newName == "" {
			return NameInvalidMsg{}
		}

		user, err := registerUser(m)
		if err != nil {
			return errMsg{err}
		}

		// Validate before resetting the session to potentially save some
		// network traffic and keep things feeling speedy.
		if !m.dbpool.ValidateName(m.newName) {
			return NameInvalidMsg{}
		}

		err = m.dbpool.SetUserName(user.ID, m.newName)
		if err == db.ErrNameTaken {
			return NameTakenMsg{}
		} else if err != nil {
			return errMsg{err}
		}

		user, err = m.dbpool.UserForKey(m.publicKey)
		if err != nil {
			return errMsg{err}
		}

		return CreateAccountMsg(user)
	}
}
