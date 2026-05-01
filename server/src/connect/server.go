package connect

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"path"

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

func loggingInterceptor() connect.Interceptor {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			rpc := path.Base(req.Spec().Procedure)
			slog.Info("RPC received", "rpc", rpc, "request", req.Any())
			resp, err := next(ctx, req)
			if err != nil {
				slog.Error("RPC error", "rpc", rpc, "err", err)
			} else {
				slog.Info("RPC completed", "rpc", rpc, "response", resp.Any())
			}
			return resp, err
		}
	})
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
	servicePath, handler := protoconnect.NewRadioServiceHandler(
		server,
		connect.WithInterceptors(loggingInterceptor(), validate.NewInterceptor()),
	)
	mux.Handle(servicePath, handler)
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
