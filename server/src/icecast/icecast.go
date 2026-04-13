package icecast

import "server/src/session"

type IcecastClient struct {
	sessionManager *session.SessionManager
}

func CreateIcecastClient(sessionManager *session.SessionManager) *IcecastClient {
	return &IcecastClient{
		sessionManager: sessionManager,
	}
}

func (*IcecastClient) StreamSessions() {
	// look at sessions in session manager,
	// get bytes needed to forward to different streams
}

func (*IcecastClient) streamSession() {

}
