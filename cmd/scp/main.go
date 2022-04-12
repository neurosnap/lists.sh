package main

// An example SCP server. This will serve files from and to ./examples/scp/testdata.

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/charmbracelet/wish"
	"github.com/gliderlabs/ssh"
	_ "github.com/lib/pq"
	"github.com/neurosnap/lists.sh/internal/db/postgres"
	"github.com/neurosnap/lists.sh/scp"
)

const host = "localhost"
const port = 23234

type SSHServer struct{}

func (me *SSHServer) authHandler(ctx ssh.Context, key ssh.PublicKey) bool {
	return true
}

func main() {
	sshServer := &SSHServer{}
	handler := &scp.DbHandler{}
	databaseUrl := os.Getenv("DATABASE_URL")
	dbh := postgres.NewDB(databaseUrl)
	defer dbh.Close()

	s, err := wish.NewServer(
		wish.WithAddress(fmt.Sprintf("%s:%d", host, port)),
		wish.WithHostKeyPath(".ssh/term_info_ed25519"),
		wish.WithPublicKeyAuth(sshServer.authHandler),
		wish.WithMiddleware(
			scp.Middleware(handler, dbh),
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
