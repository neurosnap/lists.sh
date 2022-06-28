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
	"github.com/picosh/cms"
	"github.com/picosh/cms/db/postgres"
	"github.com/picosh/send"
	"github.com/picosh/send/scp"
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

func proxyMiddleware(server *ssh.Server) error {
	cfg := internal.NewConfigSite()
	dbh := postgres.NewDB(cfg.ConfigCms)
	handler := internal.NewDbHandler(dbh)

	err := send.Middleware(handler)(server)
	if err != nil {
		return err
	}

	wish.WithMiddleware(func(sh ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			cmd := s.Command()

			if len(cmd) == 0 {
				fn := withMiddleware(
					bm.Middleware(cms.Middleware(cfg.ConfigCms)),
					lm.Middleware(),
				)
				fn(s)
				return
			}

			if cmd[0] == "scp" {
				fn := withMiddleware(scp.Middleware(handler))
				fn(s)
				return
			}
		}
	})(server)

	return nil
}

func main() {
	cfg := internal.NewConfigSite()
	logger := cfg.CreateLogger()
	host := internal.GetEnv("LISTS_HOST", "0.0.0.0")
	port := internal.GetEnv("LISTS_SSH_PORT", "2222")

	sshServer := &SSHServer{}
	s, err := wish.NewServer(
		wish.WithAddress(fmt.Sprintf("%s:%s", host, port)),
		wish.WithHostKeyPath("ssh_data/term_info_ed25519"),
		wish.WithPublicKeyAuth(sshServer.authHandler),
		proxyMiddleware,
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
