package engine

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type EventFunction func(*Engine, string) (string, error)
type CronFunction func(ticker *time.Ticker, e *Engine)
type CronFunctionContainer struct {
	Name   string
	Cron   CronFunction
	Ticker *time.Ticker
}

type Engine struct {
	frontendFiles embed.FS
	staticFiles   embed.FS
	eventMap      map[string]EventFunction
	endpointMap   map[string]string
	cronFunctions []CronFunctionContainer
	upgrader      websocket.Upgrader
	clients       map[*websocket.Conn]string
	env           string
}

func New(frontendFiles embed.FS, staticFiles embed.FS, eventMap map[string]EventFunction, endpointMap map[string]string, cronFunctions []CronFunctionContainer, env string) *Engine {
	return &Engine{
		frontendFiles: frontendFiles,
		staticFiles:   staticFiles,
		eventMap:      eventMap,
		endpointMap:   endpointMap,
		cronFunctions: cronFunctions,
		clients:       make(map[*websocket.Conn]string),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
		env: env,
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

	e.Broadcast("[broker][client_added]")

	log.Println("Client connected:", clientID)

	conn.WriteMessage(1, []byte(fmt.Sprintf("[broker_id][%s]", clientID)))

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

func (e *Engine) StartServer() error {
	for _, container := range e.cronFunctions {
		println("starting function: " + container.Name)

		go func(c CronFunctionContainer) {
			defer c.Ticker.Stop()
			c.Cron(c.Ticker, e)
		}(container)
	}

	mux := http.NewServeMux()
	//staticFS, err := fs.Sub(e.staticFiles, ".")
	//if err != nil {
	//	log.Fatal(err)
	//}

	frontendFS, err := fs.Sub(e.frontendFiles, "frontend/dist")
	if err != nil {
		log.Fatal(err)
	}

	//_ = http.FileServer(http.FS(frontendFS))
	//_ = http.FileServer(http.FS(staticFS))

	mux.Handle("/", http.FileServer(http.FS(frontendFS)))

	//mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	//	if r.URL.Path == "/" {
	//		w.Header().Set("Content-Type", "text/plain")
	//		w.Write([]byte("System Is UP!"))
	//		return
	//	}
	//	frontendFileServer.ServeHTTP(w, r)
	//})

	for endpoint, htmlFile := range e.endpointMap {
		println("setting up " + endpoint + " -> " + htmlFile)

		// Rebind htmlFile so each closure gets its own copy
		endpointCopy := endpoint
		htmlFileCopy := htmlFile

		mux.HandleFunc(endpointCopy, func(w http.ResponseWriter, r *http.Request) {
			content, err := e.staticFiles.ReadFile(htmlFileCopy)
			if err != nil {
				http.Error(w, htmlFileCopy+" not found", http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "text/html")
			w.Write(content)
		})
	}

	mux.HandleFunc("/ws", e.Handler)

	if e.env == "dev" {
		log.Printf("Starting LOCAL server on :8080")
		if err := http.ListenAndServe(":8080", mux); err != nil {
			log.Fatal("HTTP server error: ", err)
		}
	} else if e.env == "prod" {
		log.Printf("Starting PROD server with HTTPS on :443")
		log.Printf("Tic-tac-toe game: https://fund78.com:443/tictactoe")
		log.Printf("Dashboard: https://fund78.com:443/dashboard")
		log.Printf("WebSocket: wss://fund78.com:443/ws")
		if err := http.ListenAndServeTLS(":443",
			"/etc/letsencrypt/live/fund78.com/fullchain.pem",
			"/etc/letsencrypt/live/fund78.com/privkey.pem",
			mux); err != nil {
			log.Fatal("HTTPS server error: ", err)
		}
	} else {
		panic("not supported BROKER_ENV: [" + e.env + "]")
	}

	return nil
}
