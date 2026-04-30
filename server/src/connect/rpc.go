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
	slog.Info("received Ping rpc")
	return connect.NewResponse(&proto.PingResponse{Message: "Pong!"}), nil
}

func (s *Server) CreateSession(ctx context.Context, req *connect.Request[proto.CreateSessionRequest]) (*connect.Response[proto.CreateSessionResponse], error) {
	slog.Info("received CreateSession RPC", "session_id", req.Msg.SessionId)
	sessionID := session.SessionID(req.Msg.SessionId)
	if _, err := s.sessionManager.CreateSession(ctx, sessionID); err != nil {
		if errors.Is(err, session.AlreadyExistsError) {
			slog.Warn("CreateSession: session already exists", "session_id", sessionID)
			return nil, connect.NewError(connect.CodeAlreadyExists, err)
		}
		slog.Error("CreateSession: failed to create session", "session_id", sessionID, "err", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	streamURL := s.icecast.StreamURL(sessionID)
	slog.Info("CreateSession: session created", "session_id", sessionID, "stream_url", streamURL)
	return connect.NewResponse(&proto.CreateSessionResponse{StreamUrl: streamURL}), nil
}

func (s *Server) GetSession(ctx context.Context, req *connect.Request[proto.GetSessionRequest]) (*connect.Response[proto.GetSessionResponse], error) {
	slog.Info("received GetSession RPC", "session_id", req.Msg.SessionId)
	sessionID := session.SessionID(req.Msg.SessionId)
	if s.sessionManager.GetQueue(sessionID) == nil {
		slog.Error("GetSession: session not found", "session_id", sessionID)
		return nil, connect.NewError(connect.CodeNotFound, errors.New("session not found"))
	}

	streamURL := s.icecast.StreamURL(sessionID)
	slog.Info("GetSession: session found", "session_id", sessionID, "stream_url", streamURL)
	return connect.NewResponse(&proto.GetSessionResponse{StreamUrl: streamURL}), nil
}

func (s *Server) AddTrack(ctx context.Context, req *connect.Request[proto.AddTrackRequest]) (*connect.Response[proto.AddTrackResponse], error) {
	slog.Info("received AddTrack RPC", "session_id", req.Msg.SessionId, "track_url", req.Msg.TrackUrl)
	track, err := s.youtube.DownloadTrackFromURL(ctx, req.Msg.TrackUrl)
	if err != nil {
		if errors.Is(err, music.ErrInvalidURL) {
			slog.Error("AddTrack: invalid URL", "track_url", req.Msg.TrackUrl, "err", err)
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		if errors.Is(err, music.ErrVideoUnavailable) {
			slog.Error("AddTrack: video unavailable", "track_url", req.Msg.TrackUrl, "err", err)
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		slog.Error("AddTrack: failed to download track", "track_url", req.Msg.TrackUrl, "err", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	sessionID := session.SessionID(req.Msg.SessionId)
	if err := s.sessionManager.GetQueue(sessionID).Enqueue(track); err != nil {
		if errors.Is(err, session.FullQueueError) {
			slog.Error("AddTrack: queue is full", "session_id", sessionID, "err", err)
			return nil, connect.NewError(connect.CodeResourceExhausted, err)
		}
		slog.Error("AddTrack: failed to enqueue track", "session_id", sessionID, "err", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	slog.Info("AddTrack: track enqueued", "session_id", sessionID, "track_id", track.Id, "title", track.Title, "artist", track.Artist)
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
	slog.Info("received RemoveTrack RPC", "session_id", req.Msg.SessionId, "index", req.Msg.Index)
	sessionID := session.SessionID(req.Msg.SessionId)
	if err := s.sessionManager.GetQueue(sessionID).Remove(uint(req.Msg.Index)); err != nil {
		if errors.Is(err, session.BadIndexError) {
			slog.Error("RemoveTrack: invalid index", "session_id", sessionID, "index", req.Msg.Index, "err", err)
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		slog.Error("RemoveTrack: failed to remove track", "session_id", sessionID, "index", req.Msg.Index, "err", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	slog.Info("RemoveTrack: track removed", "session_id", sessionID, "index", req.Msg.Index)
	return connect.NewResponse(&proto.RemoveTrackResponse{}), nil
}

func (s *Server) SkipTrack(ctx context.Context, req *connect.Request[proto.SkipTrackRequest]) (*connect.Response[proto.SkipTrackResponse], error) {
	slog.Info("received SkipTrack RPC", "session_id", req.Msg.SessionId)
	sessionID := session.SessionID(req.Msg.SessionId)
	if err := s.sessionManager.GetQueue(sessionID).Skip(); err != nil {
		if errors.Is(err, session.EmptyQueueError) {
			slog.Error("SkipTrack: queue is empty", "session_id", sessionID, "err", err)
			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
		}
		slog.Error("SkipTrack: failed to skip track", "session_id", sessionID, "err", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	slog.Info("SkipTrack: track skipped", "session_id", sessionID)
	return connect.NewResponse(&proto.SkipTrackResponse{}), nil
}

func (s *Server) ListQueue(ctx context.Context, req *connect.Request[proto.ListQueueRequest]) (*connect.Response[proto.ListQueueResponse], error) {
	slog.Info("received ListQueue RPC", "session_id", req.Msg.SessionId)
	sessionID := session.SessionID(req.Msg.SessionId)
	tracks, err := s.sessionManager.GetQueue(sessionID).ListQueue()
	if err != nil {
		slog.Error("ListQueue: failed to list queue", "session_id", sessionID, "err", err)
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

	slog.Info("ListQueue: returning queue", "session_id", sessionID, "track_count", len(tracks))
	return connect.NewResponse(&proto.ListQueueResponse{
		Tracks: protoTracks,
	}), nil
}
