package tui

import "github.com/igor/gophkeeper/internal/clientapp/api"

type sessionLoadedMsg struct {
	token string
	err   error
}

type authDoneMsg struct {
	token string
	err   error
}

type syncDoneMsg struct {
	response api.SyncResponse
	err      error
}

type createDoneMsg struct {
	err error
}

type deleteDoneMsg struct {
	err error
}
