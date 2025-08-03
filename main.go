package main

import (
	"broker/engine"
	"broker/tictactoe"
	"bufio"
	"embed"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

//go:embed html/tic-tac-toe.html static/broker.js html/dashboard.html html/blackjack.html
var staticFiles embed.FS

var (
	games         = make(map[string]*tictactoe.Game)
	gameMux       sync.Mutex
	polygonClient *PolygonService
)

func main() {
	if runtime.GOOS == "linux" {
		err := loadEnvFile("secrets/secrets.env")
		if err != nil {
			fmt.Println("error loading env file:", err)
			return
		}
	} else {
		err := loadEnvFile("secrets/dev-secrets.env")
		if err != nil {
			fmt.Println("error loading env file:", err)
			return
		}
	}

	brokerEnv := os.Getenv("BROKER_ENV")
	polygonApi := os.Getenv("POLYGON_API")

	if brokerEnv == "" || polygonApi == "" {
		panic("BROKER_ENV and POLYGON_API is required")
	}

	polygonClient = NewPolygonService(polygonApi)

	eventMap := map[string]engine.EventFunction{
		"stock_price": stockPriceHandler,
		"watchlist":   watchListHandler,
		"connections": connectionsHandler,
		"start":       startHandler,
		"move":        playerMoveHandler,
	}
	endpointMap := map[string]string{
		"/tictactoe": "html/tic-tac-toe.html",
		"/dashboard": "html/dashboard.html",
		"/blackjack": "html/blackjack.html",
	}
	cronFunctions := []engine.CronFunctionContainer{
		{
			Name:   "Watchlist",
			Cron:   runWatchlist,
			Ticker: time.NewTicker(2 * time.Second),
		},
		{
			Name:   "Connections",
			Cron:   runGetConnections,
			Ticker: time.NewTicker(2 * time.Second),
		},
	}
	e := engine.New(staticFiles, eventMap, endpointMap, cronFunctions, brokerEnv)

	err := e.StartServer()
	if err != nil {
		panic(err)
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
	res, err := polygonClient.RunRequest(symbol)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s @ %f", symbol, res.Close), nil
}

func stockPriceHandler(e *engine.Engine, symbol string) (string, error) {
	res, err := polygonClient.RunRequest(symbol)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%f", res.Close), nil
}

func loadEnvFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}
		os.Setenv(key, value)
	}

	return scanner.Err()
}
