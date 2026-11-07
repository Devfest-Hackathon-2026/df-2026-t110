package main

import (
	"log"
	"os"

	"github.com/dvpico/Programming-Learning-AR/Server/infra/router"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/api"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/static"
	"github.com/labstack/echo/v5"
)

func main() {
	e := echo.New()

	// Unset in local development, where the Vite dev server serves the client and proxies /api.
	if dir := os.Getenv("STATIC_DIR"); dir != "" {
		e.Use(static.Middleware(dir, "/api"))
	}

	r := router.NewRouter(router.Config{})
	api.RegisterHandlersWithBaseURL(e, r, "/api")

	if err := e.Start(":8080"); err != nil {
		log.Fatal(err)
	}
}
