package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"server/src/connect"
	"server/src/db"
	"server/src/icecast"
	"server/src/music"
	"server/src/session"
	"server/src/util"
	"syscall"
	"time"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal error", "err", err)
		os.Exit(1)
	}
}

func run() error {
	slog.SetDefault(util.NewLogger())

	env, err := util.LoadEnv()
	if err != nil {
		return fmt.Errorf("failed to load config: %v", err)
	}

	db, err := db.Open(env.DB_PATH)
	if err != nil {
		return fmt.Errorf("db failed to open: %v", err)
	}
	defer db.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	server, err := connect.CreateServer(env.HOST, env.PORT, db)
	if err != nil {
		return fmt.Errorf("failed to create server: %v", err)
	}

	slog.Info("server started", "env", env, "db", db)

	y := music.NewYouTube(env.MUSIC_PATH, db)
	_, err = y.DownloadTrackFromURL(ctx, "https://www.youtube.com/watch?v=aMSXP0YV2vs")
	_, err = y.DownloadTrackFromURL(ctx, "https://youtu.be/Tib06q6wC1U?si=3XwICZzaLGGUj9Z2")
	_, err = y.DownloadTrackFromURL(ctx, "https://www.youtube.com/watch?v=utHw7pBtJM8&list=OLAK5uy_nVrWy7jixt-tF6ADSVJFCAuMh7pyqG5RY")
	t4, err := y.DownloadTrackFromURL(ctx, "https://www.youtube.com/watch?v=9No5xI-O4Es")


	sessionManager := session.CreateSessionManager()
	icecast := icecast.CreateIcecastClient(
		sessionManager,
		env.ICECAST_SERVER_HOST,
		env.ICECAST_SERVER_PORT,
		env.ICECAST_SERVER_PASSWORD,
	)

	go icecast.StreamSessions(ctx)

	sessionManager.CreateSession(ctx, "bruh")
	q := sessionManager.GetQueue("bruh")
	time.Sleep(30 * time.Second)
	q.Enqueue(t4)
	// q.Enqueue(t1)
	// q.Enqueue(t2)

	<-ctx.Done()
	slog.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(shutdownCtx)
	return nil
}
