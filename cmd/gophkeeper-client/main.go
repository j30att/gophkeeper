// Package main содержит точку входа CLI/TUI клиента GophKeeper.
package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	clientapi "github.com/igor/gophkeeper/internal/clientapp/api"
	"github.com/igor/gophkeeper/internal/clientapp/cache"
	clientconfig "github.com/igor/gophkeeper/internal/clientapp/config"
	"github.com/igor/gophkeeper/internal/clientapp/session"
	"github.com/igor/gophkeeper/internal/clientapp/tui"
)

// version подставляется во время сборки.
var version = "0.0.1"

// buildDate подставляется во время сборки.
var buildDate = "unknown"

// main запускает CLI-команду или TUI.
func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Printf("version: %s\nbuild_date: %s\n", version, buildDate)
		return
	}

	flags := flag.NewFlagSet("gophkeeper-client", flag.ExitOnError)
	serverURL := flags.String("server", "", "GophKeeper server URL")
	if err := flags.Parse(os.Args[1:]); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "parse flags: %v\n", err)
		os.Exit(1)
	}

	cfg, err := clientconfig.Load(*serverURL)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}
	apiClient, err := clientapi.NewClient(cfg.ServerURL, http.DefaultClient)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "create api client: %v\n", err)
		os.Exit(1)
	}
	sessionStore, err := session.NewStore(cfg.SessionPath)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "create session store: %v\n", err)
		os.Exit(1)
	}
	cacheStore, err := cache.NewStore(cfg.SecretsPath)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "create cache store: %v\n", err)
		os.Exit(1)
	}

	program := tea.NewProgram(tui.New(apiClient, sessionStore, cacheStore), tea.WithAltScreen())
	if _, err = program.Run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "run tui: %v\n", err)
		os.Exit(1)
	}
}
