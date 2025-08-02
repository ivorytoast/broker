package tictactoe

import (
	"broker/engine"
	"errors"
	"fmt"
	"strings"
)

type Game struct {
	Board         [9]string
	CurrentPlayer string
	Winner        string
	GameState     int
	GameID        string
}

func NewGame(gameID string) *Game {
	return &Game{
		Board:         [9]string{"-", "-", "-", "-", "-", "-", "-", "-", "-"},
		CurrentPlayer: "X",
		Winner:        "?",
		GameState:     0,
		GameID:        gameID,
	}
}

func (g *Game) ResetGame(e *engine.Engine) string {
	g.Board = [9]string{"-", "-", "-", "-", "-", "-", "-", "-", "-"}
	g.CurrentPlayer = "X"
	g.GameState = 1 // 0 = Not Started, 1 = In Progress, 2 = Done
	g.Winner = "?"

	defer e.Broadcast("[update][" + g.GameID + "," + g.FormatResponse() + "]")
	return g.FormatResponse()
}

func (g *Game) MakeMove(e *engine.Engine, move string) (string, error) {
	if g.GameState == 0 {
		println("game has not started")
		return g.FormatResponse(), nil
	}

	if g.GameState == 2 {
		println("game has ended")
		return g.FormatResponse(), nil
	}

	if len(move) != 2 {
		return "", errors.New("invalid move format")
	}

	player := string(move[0])
	pos := int(move[1] - '1') // 1-based to 0-based

	if player != g.CurrentPlayer {
		return "", fmt.Errorf("not %s's turn", player)
	}
	if pos < 0 || pos >= 9 {
		return "", fmt.Errorf("invalid position")
	}
	if g.Board[pos] != "-" {
		return "", fmt.Errorf("cell already taken")
	}

	g.Board[pos] = player

	if g.CheckWin(player) {
		g.Winner = player
		g.GameState = 2
	} else if g.IsDraw() {
		g.Winner = "T"
		g.GameState = 2
	} else {
		if g.CurrentPlayer == "X" {
			g.CurrentPlayer = "O"
		} else {
			g.CurrentPlayer = "X"
		}
	}

	defer e.Broadcast("[update][" + g.GameID + "," + g.FormatResponse() + "]")
	return g.FormatResponse(), nil
}

func (g *Game) CheckWin(player string) bool {
	winPatterns := [8][3]int{
		{0, 1, 2}, {3, 4, 5}, {6, 7, 8},
		{0, 3, 6}, {1, 4, 7}, {2, 5, 8},
		{0, 4, 8}, {2, 4, 6},
	}
	for _, pattern := range winPatterns {
		if g.Board[pattern[0]] == player &&
			g.Board[pattern[1]] == player &&
			g.Board[pattern[2]] == player {
			return true
		}
	}
	return false
}

func (g *Game) IsDraw() bool {
	for _, cell := range g.Board {
		if cell == "-" {
			return false
		}
	}
	return true
}

func (g *Game) FormatResponse() string {
	state := strings.Join(g.Board[:], ",")
	return fmt.Sprintf("%s,%s,%s,%v", state, g.CurrentPlayer, g.Winner, g.GameState)
}
