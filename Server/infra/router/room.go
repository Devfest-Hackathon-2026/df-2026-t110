package router

import (
	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/api"
	"github.com/labstack/echo/v5"
)

func (r *Router) ListRooms(ctx *echo.Context) error {
	return errNotImplemented()
}

func (r *Router) GetRoom(ctx *echo.Context, roomID api.RoomId) error {
	return errNotImplemented()
}

func (r *Router) JoinRoomEvents(ctx *echo.Context, roomID api.RoomId) error {
	return errNotImplemented()
}

func (r *Router) StartRoomMatch(ctx *echo.Context, roomID api.RoomId, matchID api.MatchId) error {
	return errNotImplemented()
}
