package ws

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func Upgrade(w http.ResponseWriter, r *http.Request) (*Conn, error) {
	if ok, msgErr := isWebSocketUpgrade(r); ok {
		return nil, fmt.Errorf("Error upgrading websocket: %v", msgErr)
	}

	acceptKey := computeAcceptKey(r.Header.Get("Sec-WebSocket-Key"))

	netconn, bufrw, err := http.NewResponseController(w).Hijack()
	if err != nil {
		return nil, fmt.Errorf("Websocket Hijack failure: %w", err)
	}

	response := "HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + acceptKey + "\r\n\r\n"

	_, err = bufrw.WriteString(response)
	if err != nil {
		return nil, fmt.Errorf("Error writing handshake response: %v\n", err)
	}

	err = bufrw.Flush()

	wsConn := &Conn{
		Conn:  netconn,
		bufrw: bufrw,
	}

	return wsConn, nil
}

// checks if request is a valid websocket upgrade request
func isWebSocketUpgrade(r *http.Request) (bool, string) {
	// http 1.1 or more are only allowed
	httpVersionMajor := r.ProtoMajor
	httpVersionMinor := r.ProtoMinor
	httpMethod := r.Method
	upgrade := strings.ToLower(r.Header.Get("Upgrade"))
	connection := strings.ToLower(r.Header.Get("Connection"))
	sec_ws_key := strings.ToLower(r.Header.Get("Sec-WebSocket-Key"))
	sec_ws_version := r.Header.Get("Sec-WebSocket-Version")

	if httpVersionMajor < 2 && httpVersionMinor < 1 {
		return false, "Upgrade request httpVersion should be more than HTTP/1.1"
	}

	if httpMethod != http.MethodGet {
		return false, "Upgrade request method should be GET"
	}

	if upgrade != "websocket" {
		return false, `Upgrade request "upgrade" header should be "websocket"`
	}

	if connection != "upgrade" {
		return false, `Upgrade request "Connection" header should be "upgrade"`
	}

	if !isValidSecKey(sec_ws_key) {
		return false, `Upgrade request security key must be valid`
	}

	if sec_ws_version != "13" {
		return false, `Upgrade request "Sec-WebSocket-Version" must be "13"`
	}

	return true, ""
}
