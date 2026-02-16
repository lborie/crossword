package main

import (
	"sync"
	"time"
)

// Player represents a connected player.
type Player struct {
	Pseudo   string    `json:"pseudo"`
	Color    string    `json:"color"`
	JoinedAt time.Time `json:"joined_at"`
}

// GameSession represents a collaborative game on a grid.
type GameSession struct {
	ID        string            `json:"id"`
	GridID    string            `json:"grid_id"`
	Players   map[string]*Player `json:"players"`
	State     [][]string        `json:"state"` // current letters [row][col]
	CreatedAt time.Time         `json:"created_at"`
	mu        sync.Mutex
}

// playerColors is the palette assigned to players in order.
var playerColors = []string{
	"#2563eb", "#dc2626", "#16a34a", "#9333ea",
	"#ea580c", "#0891b2", "#c026d3", "#ca8a04",
}

// AddPlayer adds a player to the session and returns the player.
func (g *GameSession) AddPlayer(pseudo string) *Player {
	g.mu.Lock()
	defer g.mu.Unlock()

	if p, ok := g.Players[pseudo]; ok {
		return p
	}

	p := &Player{
		Pseudo:   pseudo,
		Color:    playerColors[len(g.Players)%len(playerColors)],
		JoinedAt: time.Now(),
	}
	g.Players[pseudo] = p
	return p
}

// RemovePlayer removes a player from the session.
func (g *GameSession) RemovePlayer(pseudo string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.Players, pseudo)
}

// SetCell sets a letter at a given position. Returns false if out of bounds.
func (g *GameSession) SetCell(row, col int, value string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	if row < 0 || row >= len(g.State) || col < 0 || col >= len(g.State[0]) {
		return false
	}
	g.State[row][col] = value
	return true
}

// GetState returns a copy of the current game state.
func (g *GameSession) GetState() [][]string {
	g.mu.Lock()
	defer g.mu.Unlock()

	cp := make([][]string, len(g.State))
	for i, row := range g.State {
		cp[i] = make([]string, len(row))
		copy(cp[i], row)
	}
	return cp
}
