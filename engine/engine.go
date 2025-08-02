package engine

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/gorilla/websocket"
)

type EventFunction func(*Engine, string) (string, error)

type Engine struct {
	eventMap map[string]EventFunction
	upgrader websocket.Upgrader
	clients  map[*websocket.Conn]string
}

func New(eventMap map[string]EventFunction) *Engine {
	return &Engine{
		eventMap: eventMap,
		clients:  make(map[*websocket.Conn]string),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

func (e *Engine) GetConnections() map[*websocket.Conn]string {
	return e.clients
}

func (e *Engine) parseMessage(msg string) ([]string, error) {
	re := regexp.MustCompile(`\[[^\]]+\]`)
	matches := re.FindAllString(msg, -1)

	if matches == nil || len(matches) < 2 {
		return nil, fmt.Errorf("unexpected message format: %s", msg)
	}

	fields := make([]string, len(matches))
	for i, s := range matches {
		fields[i] = strings.Trim(s, "[]")
	}
	return fields, nil
}

func (e *Engine) ProcessMessage(msg string) (string, error) {
	fields, err := e.parseMessage(msg)
	if err != nil {
		return "", err
	}

	topic := fields[0]
	message := fields[1]

	handler, ok := e.eventMap[topic]
	if !ok {
		return "", fmt.Errorf("topic not accepted: %s", topic)
	}

	result, err := handler(e, message)
	if err != nil {
		return "", fmt.Errorf("error handling %s: %w", topic, err)
	}

	response := fmt.Sprintf("[%s][%v]", topic, result)
	return response, nil
}

func (e *Engine) Handler(w http.ResponseWriter, r *http.Request) {
	conn, err := e.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		http.Error(w, "Could not upgrade", http.StatusBadRequest)
		return
	}
	defer conn.Close()

	clientID := fmt.Sprintf("Client-%d", len(e.clients)+1)
	e.clients[conn] = clientID
	defer delete(e.clients, conn)

	log.Println("Client connected:", clientID)
	conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("hello from %s", clientID)))

	for {
		mt, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("Read error:", err)
			break
		}

		response, err := e.ProcessMessage(string(msg))
		if err != nil {
			log.Println(err)
			conn.WriteMessage(mt, []byte(err.Error()))
			continue
		}

		conn.WriteMessage(mt, []byte(response))
	}
}

func (e *Engine) Broadcast(msg string) {
	for conn := range e.clients {
		err := conn.WriteMessage(websocket.TextMessage, []byte(msg))
		if err != nil {
			log.Printf("Broadcast error: %v, removing client\n", err)
			conn.Close()
			delete(e.clients, conn)
		}
	}
}

func (e *Engine) Server(addr string) error {
	http.HandleFunc("/", e.Handler)
	log.Printf("WebSocket server listening on ws://%s\n", addr)
	return http.ListenAndServe(addr, nil)
}
