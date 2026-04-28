package connect

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"server/src/db"
	"server/src/proto"
	"server/src/proto/protoconnect"

	"connectrpc.com/connect"
	"connectrpc.com/validate"
)

type Server struct {
	protoconnect.UnimplementedRadioServiceHandler
	db *db.DB
}

func (s *Server) Ping(_ context.Context, req *connect.Request[proto.PingRequest]) (*connect.Response[proto.PingResponse], error) {
	slog.Info("received ping request")
	return connect.NewResponse(&proto.PingResponse{Message: "Pong!"}), nil
}

func CreateServer(host string, port int, db *db.DB) (*http.Server, error) {
	server := &Server{db: db}
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
