package main

import (
	"log"

	"github.com/dvpico/Programming-Learning-AR/Server/infra/router"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/api"
	"github.com/labstack/echo/v5"
)

func main() {
	e := echo.New()

	r := router.NewRouter(router.Config{})
	api.RegisterHandlersWithBaseURL(e, r, "/api")

	if err := e.Start(":8080"); err != nil {
		log.Fatal(err)
	}
}
