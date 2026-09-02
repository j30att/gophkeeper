// Package main содержит точку входа сервера GophKeeper.
package main

import (
	"context"
	"os"

	"github.com/igor/gophkeeper/internal/bootstrap"
)

// version подставляется во время сборки.
var version = "0.0.1"

// main запускает сервер GophKeeper.
func main() {
	if err := bootstrap.StartServer(context.Background(), version); err != nil {
		os.Exit(1)
	}
}
