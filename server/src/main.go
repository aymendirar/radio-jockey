package main

import (
	"log/slog"
	"os"
	"os/signal"
	"server/src/db"
	"server/src/grpc"
	"server/src/music"
	"server/src/session"
	"server/src/util"
	"syscall"
)

func main() {
	slog.SetDefault(util.NewLogger())

	env, err := util.Load()
	if err != nil {
		slog.Error("failed to load config", "err", err)
		os.Exit(1)
	}

	db, err := db.Open(env.DB_PATH)
	if err != nil {
		slog.Error("db failed to open", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	slog.Info("server started", "env", env, "db", db)

	server, err := grpc.CreateGRPCServer(env.HOST, env.PORT, db)
	if err != nil {
		slog.Error("failed to create gRPC server", "err", err)
		os.Exit(1)
	}
	defer server.Stop()

	y, err := music.CreateYouTubeClient(env.MUSIC_PATH)
	if err != nil {
		slog.Error("error creating youtube client", "err", err)
	}
	y.DownloadTrackFromURL("https://youtu.be/Tib06q6wC1U?si=3XwICZzaLGGUj9Z2")
	y.DownloadTrackFromURL("https://www.youtube.com/watch?v=aMSXP0YV2vs")
	y.DownloadTrackFromURL("https://www.youtube.com/watch?v=utHw7pBtJM8&list=OLAK5uy_nVrWy7jixt-tF6ADSVJFCAuMh7pyqG5RY")

	sessionManager := session.CreateSessionManager()
	sessionManager.CreateSession("bruh")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("shutting down")
}
