package handlers

import (
	appws "github.com/EraldCaka/pi-web/internal/ws"
	fiberws "github.com/gofiber/websocket/v2"
)

// WSHandler upgrades the HTTP connection and hands it to the hub.
type WSHandler struct {
	hub *appws.Hub
}

func NewWSHandler(hub *appws.Hub) *WSHandler {
	return &WSHandler{hub: hub}
}

// Handle is the gofiber/websocket handler. Register it with websocket.New(h.Handle).
func (h *WSHandler) Handle(c *fiberws.Conn) {
	h.hub.ServeWS(c)
}
