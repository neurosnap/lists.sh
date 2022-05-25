// Package scp provides a SCP middleware for wish.
package scp

import (
	"fmt"
	"io"
	"io/fs"
	"strconv"

	"github.com/charmbracelet/wish"
	"github.com/gliderlabs/ssh"
	"github.com/neurosnap/lists.sh/internal"
	"github.com/neurosnap/lists.sh/internal/db"
)

// CopyFromClientHandler is a handler that can be implemented to handle files
// being copied from the client to the server.
type CopyFromClientHandler interface {
	// Write should write the given file.
	Write(ssh.Session, *FileEntry, *db.User, db.DB) error
}

// Handler is a interface that can be implemented to handle both SCP
// directions.
type Handler interface {
	CopyFromClientHandler
}

// Middleware provides a wish middleware using the given CopyToClientHandler
// and CopyFromClientHandler.
func Middleware(wh CopyFromClientHandler, dbpool db.DB) wish.Middleware {
	return func(sh ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			info := GetInfo(s.Command())
			if !info.Ok {
				sh(s)
				return
			}

			if info.Recursive {
				err := fmt.Errorf("recursive not supported. try `scp ./blog/*.txt %s` instead", internal.Domain)
				errHandler(s, err)
				return
			}

			var err error
			key, err := internal.KeyText(s)
			if err != nil {
				errHandler(s, fmt.Errorf("key not found"))
				return
			}

			user, err := dbpool.FindUserForKey(s.User(), key)
			if err != nil {
				errHandler(s, err)
				return
			}

			if user.Name == "" {
				errHandler(s, fmt.Errorf("must have username set"))
				return
			}

			switch info.Op {
			case OpCopyToClient:
				err = fmt.Errorf("copying from server to client not supported")
				break
			case OpCopyFromClient:
				if wh == nil {
					err = fmt.Errorf("no handler provided for scp -t")
					break
				}
				err = copyFromClient(s, info, wh, user, dbpool)
			}
			if err != nil {
				errHandler(s, err)
				return
			}

			sh(s)
		}
	}
}

// NULL is an array with a single NULL byte.
var NULL = []byte{'\x00'}

// FileEntry is an Entry that reads from a Reader, defining a file and
// its contents.
type FileEntry struct {
	Name     string
	Filepath string
	Mode     fs.FileMode
	Size     int64
	Reader   io.Reader
	Atime    int64
	Mtime    int64
}

func (e *FileEntry) path() string { return e.Filepath }

// Write a file to the given writer.
func (e *FileEntry) Write(w io.Writer) error {
	if e.Mtime > 0 && e.Atime > 0 {
		if _, err := fmt.Fprintf(w, "T%d 0 %d 0\n", e.Mtime, e.Atime); err != nil {
			return fmt.Errorf("failed to write file: %q: %w", e.Filepath, err)
		}
	}
	if _, err := fmt.Fprintf(w, "C%s %d %s\n", octalPerms(e.Mode), e.Size, e.Name); err != nil {
		return fmt.Errorf("failed to write file: %q: %w", e.Filepath, err)
	}

	if _, err := io.Copy(w, e.Reader); err != nil {
		return fmt.Errorf("failed to read file: %q: %w", e.Filepath, err)
	}

	if _, err := w.Write(NULL); err != nil {
		return fmt.Errorf("failed to write file: %q: %w", e.Filepath, err)
	}
	return nil
}

// Op defines which kind of SCP Operation is going on.
type Op byte

const (
	// OpCopyToClient is when a file is being copied from the server to the client.
	OpCopyToClient Op = 'f'

	// OpCopyFromClient is when a file is being copied from the client into the server.
	OpCopyFromClient Op = 't'
)

// Info provides some information about the current SCP Operation.
type Info struct {
	// Ok is true if the current session is a SCP.
	Ok bool

	// Recursice is true if its a recursive SCP.
	Recursive bool

	// Path is the server path of the scp operation.
	Path string

	// Op is the SCP operation kind.
	Op Op
}

// GetInfo return information about the given command.
func GetInfo(cmd []string) Info {
	info := Info{}
	if len(cmd) == 0 || cmd[0] != "scp" {
		return info
	}

	for i, p := range cmd {
		switch p {
		case "-r":
			info.Recursive = true
		case "-f":
			info.Op = OpCopyToClient
			info.Path = cmd[i+1]
		case "-t":
			info.Op = OpCopyFromClient
			info.Path = cmd[i+1]
		}
	}

	info.Ok = true
	return info
}

func errHandler(s ssh.Session, err error) {
	_, _ = fmt.Fprintln(s.Stderr(), err)
	_ = s.Exit(1)
	_ = s.Close()
}

func octalPerms(info fs.FileMode) string {
	return "0" + strconv.FormatUint(uint64(info.Perm()), 8)
}
