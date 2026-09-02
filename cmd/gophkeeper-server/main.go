// Package main содержит точку входа сервера GophKeeper.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/igor/gophkeeper/internal/bootstrap"
)

// version подставляется во время сборки.
var version = "0.0.1"

// main запускает сервер GophKeeper.
func main() {
	if err := bootstrap.StartServer(context.Background(), version); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "server failed: %v\n", err)
		os.Exit(1)
	}
}
