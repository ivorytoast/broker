package main

import (
	"broker/engine"
	"broker/tictactoe"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

//go:embed tic-tac-toe.html broker.js index-dashboard.html
var staticFiles embed.FS

var serverType string    // Will be set via -ldflags
var polygonApiKey string // Will be set via -ldflags

var (
	games   = make(map[string]*tictactoe.Game)
	gameMux sync.Mutex
)

func main() {
	if serverType != "prod" {
		serverType = "local"
	}

	if polygonApiKey == "" {
		// polygonApiKey = ""
		panic("You must provide a Polygon API Key for development via flags or hard coded here")
	}

	InitPolygonClient(polygonApiKey)

	ticker := time.NewTicker(2 * time.Second)
	connections := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	defer connections.Stop()

	eventMap := map[string]engine.EventFunction{
		"stock_price": stockPriceHandler,
		"watchlist":   watchListHandler,
		"connections": connectionsHandler,
		"start":       startHandler,
		"move":        playerMoveHandler,
	}
	e := engine.New(eventMap)

	go runWatchlist(ticker, e)
	go runGetConnections(connections, e)

	// Create a new HTTP server
	mux := http.NewServeMux()

	// Serve static files
	staticFS, err := fs.Sub(staticFiles, ".")
	if err != nil {
		log.Fatal(err)
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte("Hello from backend!"))
			return
		}
		if r.URL.Path == "/tictactoe" {
			content, err := staticFiles.ReadFile("tic-tac-toe.html")
			if err != nil {
				http.Error(w, "Tic-tac-toe game not found", http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "text/html")
			w.Write(content)
			return
		}
		http.FileServer(http.FS(staticFS)).ServeHTTP(w, r)
	})

	mux.HandleFunc("/dashboard", func(w http.ResponseWriter, r *http.Request) {
		content, err := staticFiles.ReadFile("index-dashboard.html")
		if err != nil {
			http.Error(w, "Dashboard not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.Write(content)
	})

	mux.HandleFunc("/ws", e.Handler)

	if serverType == "local" {
		log.Printf("Starting LOCAL server on :8080")
		if err := http.ListenAndServe(":8080", mux); err != nil {
			log.Fatal("HTTP server error: ", err)
		}
	} else if serverType == "prod" {
		log.Printf("Starting PROD server with HTTPS on :443")
		log.Printf("Tic-tac-toe game: https://chalkedup.io:443/tictactoe")
		log.Printf("Dashboard: https://chalkedup.io:443/dashboard")
		log.Printf("WebSocket: wss://chalkedup.io:443/ws")
		if err := http.ListenAndServeTLS(":443",
			"/etc/letsencrypt/live/chalkedup.io/fullchain.pem",
			"/etc/letsencrypt/live/chalkedup.io/privkey.pem",
			mux); err != nil {
			log.Fatal("HTTPS server error: ", err)
		}
	}
}

func startHandler(e *engine.Engine, gameID string) (string, error) {
	println("got start with name: " + gameID)
	// 1,2,3,4,5,6,7,8,9,Player,Winner,GameState
	gameMux.Lock()
	game, ok := games[gameID]
	if !ok {
		game = tictactoe.NewGame(gameID)
		games[gameID] = game
	}
	gameMux.Unlock()
	return game.ResetGame(e), nil
}

func playerMoveHandler(e *engine.Engine, moveInput string) (string, error) {
	println("got move: " + moveInput)
	elements := strings.Split(moveInput, ",")
	gameID := elements[0]
	move := elements[1]

	gameMux.Lock()
	game, ok := games[gameID]
	if !ok {
		println("game not found: " + gameID)
		return game.ResetGame(e), nil
	}
	gameMux.Unlock()

	return game.MakeMove(e, move)
}

func runGetConnections(ticker *time.Ticker, e *engine.Engine) {
	for range ticker.C {
		connections := e.GetConnections()

		var connList []string
		for _, name := range connections {
			connList = append(connList, name)
		}

		connStr := strings.Join(connList, ", ")
		msg := fmt.Sprintf("[connections][%s]", connStr)

		response, err := e.ProcessMessage(msg)
		if err != nil {
			fmt.Println("Error:", err)
		} else {
			fmt.Println("Broadcasting:", response)
			e.Broadcast(response)
		}
	}
}

func runWatchlist(ticker *time.Ticker, e *engine.Engine) {
	symbols := []string{"AAPL", "MSFT", "GOOG", "TSLA"}
	i := 0
	for range ticker.C {
		symbol := symbols[i%len(symbols)]
		msg := fmt.Sprintf("[watchlist][%s]", symbol)

		response, err := e.ProcessMessage(msg)
		if err != nil {
			fmt.Println("Error:", err)
		} else {
			fmt.Println("Broadcasting:", response)
			e.Broadcast(response)
		}
		i++
	}
}

func connectionsHandler(e *engine.Engine, connections string) (string, error) {
	return fmt.Sprintf("%s", connections), nil
}

func watchListHandler(e *engine.Engine, symbol string) (string, error) {
	res, err := runPolygonRequest(symbol)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s @ %f", symbol, res.Close), nil
}

func stockPriceHandler(e *engine.Engine, symbol string) (string, error) {
	res, err := runPolygonRequest(symbol)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%f", res.Close), nil
}
