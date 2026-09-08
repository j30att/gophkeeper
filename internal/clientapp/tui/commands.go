package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"

	"github.com/igor/gophkeeper/internal/clientapp/api"
	"github.com/igor/gophkeeper/internal/clientapp/session"
)

func loadSessionCmd(store SessionStore) tea.Cmd {
	return func() tea.Msg {
		session, err := store.Load()
		if err != nil {
			return sessionLoadedMsg{err: err}
		}
		return sessionLoadedMsg{token: session.Token}
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

func deleteCmd(client APIClient, token string, id uuid.UUID) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return deleteDoneMsg{err: client.DeleteSecret(ctx, token, id)}
	}
}
