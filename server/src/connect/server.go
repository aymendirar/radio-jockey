package connect

import (
	"fmt"
	"log/slog"
	"net/http"

	"server/src/icecast"
	"server/src/music"
	"server/src/proto/protoconnect"
	"server/src/session"

	"connectrpc.com/connect"
	"connectrpc.com/validate"
)

type Server struct {
	protoconnect.UnimplementedRadioServiceHandler
	sessionManager *session.SessionManager
	youtube        *music.YouTube
	icecast        *icecast.IcecastClient
}

func CreateServer(
	host string,
	port int,
	sessionManager *session.SessionManager,
	youtube *music.YouTube,
	icecast *icecast.IcecastClient) (*http.Server, error) {
	server := &Server{
		sessionManager: sessionManager,
		youtube:        youtube,
		icecast:        icecast,
	}
	mux := http.NewServeMux()
	path, handler := protoconnect.NewRadioServiceHandler(
		server,
		connect.WithInterceptors(validate.NewInterceptor()),
	)
	mux.Handle(path, handler)
	p := new(http.Protocols)
	p.SetHTTP1(true)
	p.SetUnencryptedHTTP2(true)
	s := &http.Server{
		Addr:      fmt.Sprintf("%s:%d", host, port),
		Handler:   mux,
		Protocols: p,
	}

	go func() {
		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("connect server error", "error", err)
		}
	}()

	return s, nil
}
