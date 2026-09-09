package tui

import (
	"github.com/igor/gophkeeper/internal/clientapp/api"
	"github.com/igor/gophkeeper/internal/clientapp/cache"
)

type sessionLoadedMsg struct {
	token string
	err   error
}

type cacheLoadedMsg struct {
	cache cache.Cache
	err   error
}

type cacheSavedMsg struct {
	err error
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

type updateDoneMsg struct {
	err error
}

type createBlobDoneMsg struct {
	err error
}

type downloadBlobDoneMsg struct {
	path string
	err  error
}

type updateBlobDoneMsg struct {
	err error
}

type logoutDoneMsg struct {
	err error
}

type deleteDoneMsg struct {
	err error
}
