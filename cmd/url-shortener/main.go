package main

import (
	"fmt"
	"url-shortener/internal/config"
)

func main() {
	config := config.MustLoad()

	fmt.Println(config)

	// TODO: init logger: slog

	// TODO: init storage: sqlite

	// TODO: init router: chi, render

	// TODO: init server
}
