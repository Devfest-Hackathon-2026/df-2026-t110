package main

import (
	"log"

	"github.com/dvpico/Programming-Learning-AR/Server/infra/pubsub"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/router"
	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/api"
	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
	"github.com/labstack/echo/v5"
)

func main() {
	e := echo.New()

	roomBroker := pubsub.NewRoomBroker(0)
	roomEvents := usecase.NewRoomEvents(roomBroker)

	r := router.NewRouter(roomEvents, router.Config{})
	api.RegisterHandlersWithBaseURL(e, r, "/api")

	if err := e.Start(":8080"); err != nil {
		log.Fatal(err)
	}
}
