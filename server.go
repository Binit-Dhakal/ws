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

func HandleConnection(wsConn *Conn) {
	pingTicker := time.NewTicker(pingPeriod)
	defer pingTicker.Stop()

	pingChan := make(chan bool, 1)
	incomingFramesChan := make(chan *Frame, 5)
	readErrChan := make(chan error, 1)

	// Goroutine to send periodic message
	go func() {
		for range pingTicker.C {
			select {
			case pingChan <- true:
			default:
				// channel full skip this beat
			}
		}
	}()

	go func() {
		for {
			frame, err := wsConn.ReadFrame()
			if err != nil {
				select {
				case readErrChan <- err:
				default:
				}
				return
			}

			select {
			case incomingFramesChan <- frame:
			// Successfully sent the frame
			default:
				fmt.Println("Warning: Incoming frame channel full. dropping frame")
			}
		}
	}()

	for {
		select {
		case <-pingChan:
			pingPayload := []byte("pong")
			if err := wsConn.WritePingFrame(pingPayload); err != nil {
				fmt.Printf("Error sending ping: %v\n", err)
				return
			}
		case frame := <-incomingFramesChan:
			if frame == nil {
				continue
			}

			switch frame.OpCode {
			case OpcodeText:
				message := string(frame.Payload)
				fmt.Printf("Received text message: %s\n", message)

				response := fmt.Sprintf("Echo: %s", message)
				if err := wsConn.WriteTextFrame(response); err != nil {
					fmt.Printf("Error sending response: %v\n", err)
					return
				}

			case OpcodeClose:
				fmt.Println("Received close frame")
				wsConn.WriteCloseFrame()
				return

			case OpcodePing:
				// less common but possible
				fmt.Println("Received ping frame from client")
				wsConn.WritePongFrame(frame.Payload)

			case OpcodePong:
				fmt.Println("Received pong frame from client")

			default:
				fmt.Printf("Received unsupported frame type: %d\n", frame.OpCode)
			}
		case err := <-readErrChan:
			if err == io.EOF {
				fmt.Println("Client disconnected gracefully (EOF on read)")
			} else {
				fmt.Println("Error from reader goroutine:", err)
			}
			return
		}
	}
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
