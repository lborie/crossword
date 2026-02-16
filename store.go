package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// Store holds all grids and game sessions in memory.
type Store struct {
	mu    sync.RWMutex
	grids map[string]*Grid
	games map[string]*GameSession
}

// NewStore creates an empty store.
func NewStore() *Store {
	return &Store{
		grids: make(map[string]*Grid),
		games: make(map[string]*GameSession),
	}
}

// SaveGrid persists a grid and returns it with a generated ID.
func (s *Store) SaveGrid(g *Grid) *Grid {
	g.ID = generateID()
	g.CreatedAt = time.Now()

	s.mu.Lock()
	s.grids[g.ID] = g
	s.mu.Unlock()

	return g
}

// GetGrid returns a grid by ID, or nil if not found.
func (s *Store) GetGrid(id string) *Grid {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.grids[id]
}

// ListGrids returns all grids, most recent first.
func (s *Store) ListGrids() []*Grid {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := make([]*Grid, 0, len(s.grids))
	for _, g := range s.grids {
		list = append(list, g)
	}
	// Sort by CreatedAt descending (simple insertion, small N).
	for i := 1; i < len(list); i++ {
		for j := i; j > 0 && list[j].CreatedAt.After(list[j-1].CreatedAt); j-- {
			list[j], list[j-1] = list[j-1], list[j]
		}
	}
	return list
}

// CreateGame creates a new game session for a given grid.
// Returns an error if the grid does not exist.
func (s *Store) CreateGame(gridID string) (*GameSession, error) {
	s.mu.RLock()
	grid := s.grids[gridID]
	s.mu.RUnlock()

	if grid == nil {
		return nil, fmt.Errorf("grid not found: %s", gridID)
	}

	// Initialize empty state matching grid dimensions.
	state := make([][]string, grid.Rows)
	for i := range state {
		state[i] = make([]string, grid.Cols)
	}

	game := &GameSession{
		ID:        generateID(),
		GridID:    gridID,
		Players:   make(map[string]*Player),
		State:     state,
		CreatedAt: time.Now(),
	}

	s.mu.Lock()
	s.games[game.ID] = game
	s.mu.Unlock()

	return game, nil
}

// GetGame returns a game session by ID, or nil if not found.
func (s *Store) GetGame(id string) *GameSession {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.games[id]
}

// ListGames returns all game sessions.
func (s *Store) ListGames() []*GameSession {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := make([]*GameSession, 0, len(s.games))
	for _, g := range s.games {
		list = append(list, g)
	}
	return list
}

func generateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}
