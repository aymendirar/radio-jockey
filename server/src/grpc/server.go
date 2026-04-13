package grpc

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"server/src/db"
	pb "server/src/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type server struct {
	pb.UnimplementedDiscordJockeyServiceServer
}

func (s *server) Ping(_ context.Context, in *pb.PingRequest) (*pb.PingResponse, error) {
	slog.Info("received ping request")
	return &pb.PingResponse{Message: "Pong"}, nil
}

func CreateGRPCServer(host string, port int, db *db.DB) (*grpc.Server, error) {
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return nil, err
	}

	s := grpc.NewServer()
	reflection.Register(s)
	pb.RegisterDiscordJockeyServiceServer(s, &server{})
	slog.Info("server listening", "addr", lis.Addr())

	go func() {
		if err := s.Serve(lis); err != nil {
			slog.Error("gRPC server failed", "err", err)
		}
	}()

	return s, nil
}
