package tui

import (
	"context"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"

	"github.com/igor/gophkeeper/internal/clientapp/api"
	"github.com/igor/gophkeeper/internal/clientapp/cache"
	"github.com/igor/gophkeeper/internal/clientapp/session"
)

func loadSessionCmd(store SessionStore) tea.Cmd {
	return func() tea.Msg {
		localSession, err := store.Load()
		if err != nil {
			return sessionLoadedMsg{err: err}
		}
		return sessionLoadedMsg{token: localSession.Token}
	}
}

func loadCacheCmd(store CacheStore) tea.Cmd {
	return func() tea.Msg {
		if store == nil {
			return cacheLoadedMsg{}
		}
		localCache, err := store.Load()
		return cacheLoadedMsg{cache: localCache, err: err}
	}
}

func saveCacheCmd(store CacheStore, localCache cache.Cache) tea.Cmd {
	return func() tea.Msg {
		if store == nil {
			return cacheSavedMsg{}
		}
		return cacheSavedMsg{err: store.Save(localCache)}
	}
}

func saveSessionCmd(store SessionStore, token string, next tea.Cmd) tea.Cmd {
	return tea.Batch(
		func() tea.Msg {
			if err := store.Save(session.Session{Token: token}); err != nil {
				return authDoneMsg{err: err}
			}
			return nil
		},
		next,
	)
}

func logoutCmd(store SessionStore, cacheStore CacheStore) tea.Cmd {
	return func() tea.Msg {
		if store != nil {
			if err := store.Delete(); err != nil {
				return logoutDoneMsg{err: err}
			}
		}
		if cacheStore != nil {
			if err := cacheStore.Delete(); err != nil {
				return logoutDoneMsg{err: err}
			}
		}
		return logoutDoneMsg{}
	}
}

func authCmd(client APIClient, login string, password string, register bool) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if register {
			if _, err := client.Register(ctx, login, password); err != nil {
				return authDoneMsg{err: err}
			}
		}
		token, err := client.Login(ctx, login, password)
		return authDoneMsg{token: token, err: err}
	}
}

func syncCmd(client APIClient, token string, since *time.Time) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		response, err := client.Sync(ctx, token, since)
		return syncDoneMsg{response: response, err: err}
	}
}

func createCmd(client APIClient, token string, input api.SecretInput) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, err := client.CreateSecret(ctx, token, input)
		return createDoneMsg{err: err}
	}
}

func updateCmd(client APIClient, token string, id uuid.UUID, input api.SecretInput) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, err := client.UpdateSecret(ctx, token, id, input)
		return updateDoneMsg{err: err}
	}
}

func createBlobCmd(client APIClient, token string, input api.BlobSecretInput, closeFile func() error) tea.Cmd {
	return func() tea.Msg {
		defer func() {
			if closeFile != nil {
				_ = closeFile()
			}
		}()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_, err := client.CreateBlobSecret(ctx, token, input)
		return createBlobDoneMsg{err: err}
	}
}

func downloadBlobCmd(client APIClient, token string, id uuid.UUID, path string, closeFile func() error, file *os.File) tea.Cmd {
	return func() tea.Msg {
		defer func() {
			if closeFile != nil {
				_ = closeFile()
			}
		}()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		err := client.DownloadBlobContent(ctx, token, id, file)
		return downloadBlobDoneMsg{path: path, err: err}
	}
}

func updateBlobCmd(client APIClient, token string, id uuid.UUID, input api.BlobContentInput, closeFile func() error) tea.Cmd {
	return func() tea.Msg {
		defer func() {
			if closeFile != nil {
				_ = closeFile()
			}
		}()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_, err := client.UpdateBlobContent(ctx, token, id, input)
		return updateBlobDoneMsg{err: err}
	}
}

func deleteCmd(client APIClient, token string, id uuid.UUID) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return deleteDoneMsg{id: id, err: client.DeleteSecret(ctx, token, id)}
	}
}
