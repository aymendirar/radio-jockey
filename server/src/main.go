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
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	env, err := util.LoadEnv()
	if err != nil {
		return fmt.Errorf("failed to load config: %v", err)
	}
	slog.Info("config loaded", "host", env.HOST, "port", env.PORT, "db", env.DB_PATH, "music", env.MUSIC_PATH)

	db, err := db.Open(env.DB_PATH)
	if err != nil {
		return fmt.Errorf("db failed to open: %v", err)
	}
	defer db.Close()
	slog.Info("database opened", "path", env.DB_PATH)

	youtube := music.NewYouTube(env.MUSIC_PATH, db)
	slog.Info("youtube client created", "music_path", env.MUSIC_PATH)

	sessionManager := session.CreateSessionManager()
	slog.Info("session manager created")

	icecast := icecast.CreateIcecastClient(
		sessionManager,
		env.ICECAST_SERVER_HOST,
		env.ICECAST_SERVER_PORT,
		env.ICECAST_SERVER_PASSWORD,
		env.STREAM_BASE_URL,
	)
	slog.Info("icecast client created", "host", env.ICECAST_SERVER_HOST, "port", env.ICECAST_SERVER_PORT, "stream_base_url", env.STREAM_BASE_URL)

	server, err := connect.CreateServer(env.HOST, env.PORT, sessionManager, youtube, icecast)
	if err != nil {
		return fmt.Errorf("failed to create server: %v", err)
	}
	slog.Info("connect server started", "addr", fmt.Sprintf("%s:%d", env.HOST, env.PORT))

	go icecast.StreamSessions(ctx)
	slog.Info("icecast session streaming started")

	<-ctx.Done()
	slog.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(shutdownCtx)
	slog.Info("shutdown complete")
	return nil
}
