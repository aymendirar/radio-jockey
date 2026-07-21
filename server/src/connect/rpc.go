package connect

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"server/src/db"
	"server/src/music"
	"server/src/proto"
	"server/src/session"

	"connectrpc.com/connect"
)

// rpcErrorToCode maps known domain errors to their ConnectRPC status codes.
// Falls back to CodeInternal for unrecognized errors.
func rpcErrorToCode(err error) connect.Code {
	switch {
	case errors.Is(err, session.SessionNotFoundError):
		return connect.CodeNotFound
	case errors.Is(err, session.AlreadyExistsError):
		return connect.CodeAlreadyExists
	case errors.Is(err, session.TooManySessionsError):
		return connect.CodeResourceExhausted
	case errors.Is(err, session.FullQueueError):
		return connect.CodeResourceExhausted
	case errors.Is(err, session.EmptyQueueError):
		return connect.CodeFailedPrecondition
	case errors.Is(err, session.BadIndexError):
		return connect.CodeInvalidArgument
	case errors.Is(err, music.ErrInvalidURL):
		return connect.CodeInvalidArgument
	case errors.Is(err, music.ErrVideoUnavailable):
		return connect.CodeNotFound
	case errors.Is(err, sql.ErrNoRows):
		return connect.CodeNotFound
	default:
		return connect.CodeInternal
	}
}

func trackToProto(t *db.Track) *proto.Track {
	return &proto.Track{
		Id:          t.ID,
		Source:      t.Source,
		SourceId:    t.SourceID,
		Title:       t.Title,
		Artist:      t.Artist,
		Duration:    t.Duration,
		AlbumArtUrl: t.AlbumArtURL,
	}
}

func (s *Server) Ping(_ context.Context, req *connect.Request[proto.PingRequest]) (*connect.Response[proto.PingResponse], error) {
	return connect.NewResponse(&proto.PingResponse{Message: "Pong!"}), nil
}

func (s *Server) RequestNonce(context.Context, *connect.Request[proto.RequestNonceRequest]) (*connect.Response[proto.RequestNonceResponse], error) {
	nonce, err := s.auth.IssueNonce()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("bad"))
	}
	return connect.NewResponse(&proto.RequestNonceResponse{Nonce: nonce}), nil
}

func (s *Server) RespondNonce(ctx context.Context, req *connect.Request[proto.RespondNonceRequest]) (*connect.Response[proto.RespondNonceResponse], error) {
	if !s.auth.ConsumeNonce(req.Msg.PassKey) {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("bad"))
	}

	token := s.auth.NewToken()
	return connect.NewResponse(&proto.RespondNonceResponse{AuthToken: token}), nil
}

func (s *Server) DeleteSessionAuth(ctx context.Context, req *connect.Request[proto.DeleteSessionAuthRequest]) (*connect.Response[proto.DeleteSessionAuthResponse], error) {
	sessionID := session.SessionID(req.Msg.SessionId)
	if err := s.sessionManager.DeleteSession(ctx, sessionID); err != nil {
		return nil, connect.NewError(rpcErrorToCode(err), err)
	}
	return connect.NewResponse(&proto.DeleteSessionAuthResponse{}), nil
}

func (s *Server) CreateSession(ctx context.Context, req *connect.Request[proto.CreateSessionRequest]) (*connect.Response[proto.CreateSessionResponse], error) {
	sessionID := session.SessionID(req.Msg.SessionId)

	var archiveID *int64
	if req.Msg.Archive {
		archive, err := s.db.CreateSessionArchive(ctx, req.Msg.SessionId)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		archiveID = &archive.ID
	}

	ready, err := s.sessionManager.CreateSession(ctx, sessionID, archiveID)
	if err != nil {
		return nil, connect.NewError(rpcErrorToCode(err), err)
	}

	// block on ready
	select {
	case err := <-ready:
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	case <-ctx.Done():
		return nil, connect.NewError(connect.CodeDeadlineExceeded, ctx.Err())
	}

	streamURL := s.icecast.StreamURL(sessionID)
	return connect.NewResponse(&proto.CreateSessionResponse{StreamUrl: streamURL}), nil
}

func (s *Server) GetSession(ctx context.Context, req *connect.Request[proto.GetSessionRequest]) (*connect.Response[proto.GetSessionResponse], error) {
	sessionID := session.SessionID(req.Msg.SessionId)
	if _, err := s.sessionManager.GetQueue(sessionID); err != nil {
		return nil, connect.NewError(rpcErrorToCode(err), err)
	}

	streamURL := s.icecast.StreamURL(sessionID)
	return connect.NewResponse(&proto.GetSessionResponse{StreamUrl: streamURL}), nil
}

func (s *Server) ListSessions(ctx context.Context, req *connect.Request[proto.ListSessionsRequest]) (*connect.Response[proto.ListSessionsResponse], error) {
	sessionIDs := s.sessionManager.ListSessions()
	sessions := make([]*proto.SessionInfo, len(sessionIDs))
	for i, id := range sessionIDs {
		sessions[i] = &proto.SessionInfo{
			SessionId: string(id),
			StreamUrl: s.icecast.StreamURL(id),
		}
	}

	return connect.NewResponse(&proto.ListSessionsResponse{Sessions: sessions}), nil
}

func (s *Server) AddTrack(ctx context.Context, req *connect.Request[proto.AddTrackRequest]) (*connect.Response[proto.AddTrackResponse], error) {
	track, err := s.youtube.DownloadTrackFromURL(ctx, req.Msg.TrackUrl)
	if err != nil {
		return nil, connect.NewError(rpcErrorToCode(err), err)
	}

	sessionID := session.SessionID(req.Msg.SessionId)
	queue, err := s.sessionManager.GetQueue(sessionID)
	if err != nil {
		return nil, connect.NewError(rpcErrorToCode(err), err)
	}
	if err := queue.Enqueue(track); err != nil {
		return nil, connect.NewError(rpcErrorToCode(err), err)
	}

	if err := s.cache.Touch(ctx, track, s.sessionManager.InUseTrackIDs()); err != nil {
		slog.Error("cache touch failed", "err", err)
	}

	return connect.NewResponse(&proto.AddTrackResponse{
		Track: trackToProto(track),
	}), nil
}

func (s *Server) RemoveTrack(ctx context.Context, req *connect.Request[proto.RemoveTrackRequest]) (*connect.Response[proto.RemoveTrackResponse], error) {
	sessionID := session.SessionID(req.Msg.SessionId)
	queue, err := s.sessionManager.GetQueue(sessionID)
	if err != nil {
		return nil, connect.NewError(rpcErrorToCode(err), err)
	}
	if req.Msg.Index == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("cannot remove the currently playing track"))
	}
	if err := queue.Remove(uint(req.Msg.Index)); err != nil {
		return nil, connect.NewError(rpcErrorToCode(err), err)
	}

	return connect.NewResponse(&proto.RemoveTrackResponse{}), nil
}

func (s *Server) SkipTrack(ctx context.Context, req *connect.Request[proto.SkipTrackRequest]) (*connect.Response[proto.SkipTrackResponse], error) {
	sessionID := session.SessionID(req.Msg.SessionId)
	queue, err := s.sessionManager.GetQueue(sessionID)
	if err != nil {
		return nil, connect.NewError(rpcErrorToCode(err), err)
	}
	if err := queue.Skip(); err != nil {
		return nil, connect.NewError(rpcErrorToCode(err), err)
	}

	return connect.NewResponse(&proto.SkipTrackResponse{}), nil
}

func (s *Server) ListQueue(ctx context.Context, req *connect.Request[proto.ListQueueRequest]) (*connect.Response[proto.ListQueueResponse], error) {
	sessionID := session.SessionID(req.Msg.SessionId)
	queue, err := s.sessionManager.GetQueue(sessionID)
	if err != nil {
		return nil, connect.NewError(rpcErrorToCode(err), err)
	}

	tracks, err := queue.ListQueue()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoTracks := make([]*proto.Track, len(tracks))
	for i, t := range tracks {
		protoTracks[i] = trackToProto(t)
	}

	return connect.NewResponse(&proto.ListQueueResponse{Tracks: protoTracks}), nil
}

func (s *Server) ListSessionArchives(ctx context.Context, req *connect.Request[proto.ListSessionArchivesRequest]) (*connect.Response[proto.ListSessionArchivesResponse], error) {
	archives, err := s.db.ListSessionArchives(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoArchives := make([]*proto.SessionArchiveInfo, len(archives))
	for i, archive := range archives {
		protoArchives[i] = &proto.SessionArchiveInfo{
			Id:        archive.ID,
			SessionId: archive.SessionID,
			CreatedAt: archive.CreatedAt,
		}
	}

	return connect.NewResponse(&proto.ListSessionArchivesResponse{Archives: protoArchives}), nil
}

func (s *Server) GetSessionArchive(ctx context.Context, req *connect.Request[proto.GetSessionArchiveRequest]) (*connect.Response[proto.GetSessionArchiveResponse], error) {
	archive, err := s.db.GetSessionArchive(ctx, req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(rpcErrorToCode(err), err)
	}

	tracks, err := s.db.ListSessionArchiveTracks(ctx, archive.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoTracks := make([]*proto.Track, len(tracks))
	for i, t := range tracks {
		protoTracks[i] = trackToProto(t)
	}

	return connect.NewResponse(&proto.GetSessionArchiveResponse{
		Id:        archive.ID,
		SessionId: archive.SessionID,
		CreatedAt: archive.CreatedAt,
		Tracks:    protoTracks,
	}), nil
}

func (s *Server) DeleteSessionArchive(ctx context.Context, req *connect.Request[proto.DeleteSessionArchiveRequest]) (*connect.Response[proto.DeleteSessionArchiveResponse], error) {
	if _, err := s.db.GetSessionArchive(ctx, req.Msg.Id); err != nil {
		return nil, connect.NewError(rpcErrorToCode(err), err)
	}

	if err := s.db.DeleteSessionArchive(ctx, req.Msg.Id); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&proto.DeleteSessionArchiveResponse{}), nil
}
