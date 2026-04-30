package connect

import (
	"context"
	"errors"
	"log/slog"
	"server/src/music"
	"server/src/proto"
	"server/src/session"

	"connectrpc.com/connect"
)

func (s *Server) Ping(_ context.Context, req *connect.Request[proto.PingRequest]) (*connect.Response[proto.PingResponse], error) {
	slog.Info("received ping request")
	return connect.NewResponse(&proto.PingResponse{Message: "Pong!"}), nil
}

func (s *Server) CreateSession(ctx context.Context, req *connect.Request[proto.CreateSessionRequest]) (*connect.Response[proto.CreateSessionResponse], error) {
	sessionID := session.SessionID(req.Msg.SessionId)
	if _, err := s.sessionManager.CreateSession(ctx, sessionID); err != nil {
		if errors.Is(err, session.AlreadyExistsError) {
			return nil, connect.NewError(connect.CodeAlreadyExists, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&proto.CreateSessionResponse{
		StreamUrl: s.icecast.StreamURL(sessionID),
	}), nil
}

func (s *Server) AddTrack(ctx context.Context, req *connect.Request[proto.AddTrackRequest]) (*connect.Response[proto.AddTrackResponse], error) {
	track, err := s.youtube.DownloadTrackFromURL(ctx, req.Msg.TrackUrl)
	if err != nil {
		if errors.Is(err, music.ErrInvalidURL) {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	sessionID := session.SessionID(req.Msg.SessionId)
	if err := s.sessionManager.GetQueue(sessionID).Enqueue(track); err != nil {
		if errors.Is(err, session.FullQueueError) {
			return nil, connect.NewError(connect.CodeResourceExhausted, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&proto.AddTrackResponse{
		Track: &proto.Track{
			Id:       track.Id,
			Source:   track.Source,
			SourceId: track.SourceId,
			Title:    track.Title,
			Artist:   track.Artist,
			Duration: track.Duration,
		},
	}), nil
}

func (s *Server) RemoveTrack(ctx context.Context, req *connect.Request[proto.RemoveTrackRequest]) (*connect.Response[proto.RemoveTrackResponse], error) {
	sessionID := session.SessionID(req.Msg.SessionId)
	if err := s.sessionManager.GetQueue(sessionID).Remove(uint(req.Msg.Index)); err != nil {
		if errors.Is(err, session.BadIndexError) {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&proto.RemoveTrackResponse{}), nil
}

func (s *Server) SkipTrack(ctx context.Context, req *connect.Request[proto.SkipTrackRequest]) (*connect.Response[proto.SkipTrackResponse], error) {
	sessionID := session.SessionID(req.Msg.SessionId)
	if err := s.sessionManager.GetQueue(sessionID).Skip(); err != nil {
		if errors.Is(err, session.EmptyQueueError) {
			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&proto.SkipTrackResponse{}), nil
}

func (s *Server) ListQueue(ctx context.Context, req *connect.Request[proto.ListQueueRequest]) (*connect.Response[proto.ListQueueResponse], error) {
	sessionID := session.SessionID(req.Msg.SessionId)
	tracks, err := s.sessionManager.GetQueue(sessionID).ListQueue()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoTracks := make([]*proto.Track, len(tracks))
	for i, t := range tracks {
		protoTracks[i] = &proto.Track{
			Id:       t.Id,
			Source:   t.Source,
			SourceId: t.SourceId,
			Title:    t.Title,
			Artist:   t.Artist,
			Duration: t.Duration,
		}
	}

	return connect.NewResponse(&proto.ListQueueResponse{
		Tracks: protoTracks,
	}), nil
}
