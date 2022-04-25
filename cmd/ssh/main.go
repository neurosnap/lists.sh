package main

// An example Bubble Tea server. This will put an ssh session into alt screen
// and continually print up to date terminal information.

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/charmbracelet/wish"
	bm "github.com/charmbracelet/wish/bubbletea"
	lm "github.com/charmbracelet/wish/logging"
	"github.com/gliderlabs/ssh"
	"github.com/neurosnap/lists.sh/internal"
	"github.com/neurosnap/lists.sh/internal/cms"
	"github.com/neurosnap/lists.sh/internal/db/postgres"
	"github.com/neurosnap/lists.sh/internal/scp"
)

type SSHServer struct{}

func (me *SSHServer) authHandler(ctx ssh.Context, key ssh.PublicKey) bool {
	return true
}

func withMiddleware(mw ...wish.Middleware) ssh.Handler {
	h := func(s ssh.Session) {}
	for _, m := range mw {
		h = m(h)
	}
	return h
}

func proxyMiddleware() wish.Middleware {
	return func(sh ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			cmd := s.Command()

			if len(cmd) == 0 {
				fn := withMiddleware(
					bm.Middleware(cms.Handler),
					lm.Middleware(),
				)
				fn(s)
				return
			}

			if cmd[0] == "scp" {
				handler := &scp.DbHandler{}
				dbh := postgres.NewDB()
				fn := withMiddleware(scp.Middleware(handler, dbh))
				fn(s)
				return
			}
		}
	}
}

func main() {
	logger := internal.CreateLogger()
	host := internal.GetEnv("LISTS_HOST", "0.0.0.0")
	port := internal.GetEnv("LISTS_SSH_PORT", "2222")

	sshServer := &SSHServer{}
	s, err := wish.NewServer(
		wish.WithAddress(fmt.Sprintf("%s:%s", host, port)),
		wish.WithHostKeyPath("ssh_data/term_info_ed25519"),
		wish.WithPublicKeyAuth(sshServer.authHandler),
		wish.WithMiddleware(proxyMiddleware()),
	)
	if err != nil {
		logger.Fatal(err)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	logger.Infof("Starting SSH server on %s:%s", host, port)
	go func() {
		if err = s.ListenAndServe(); err != nil {
			logger.Fatal(err)
		}
	}()

	<-done
	logger.Info("Stopping SSH server")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer func() { cancel() }()
	if err := s.Shutdown(ctx); err != nil {
		logger.Fatal(err)
	}
}
