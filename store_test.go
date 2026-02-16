package main

import (
	"sync"
	"testing"
)

func newTestGrid(rows, cols int) *Grid {
	cells := make([][]Cell, rows)
	for i := range cells {
		cells[i] = make([]Cell, cols)
	}
	return &Grid{Rows: rows, Cols: cols, Cells: cells}
}

func TestSaveAndGetGrid(t *testing.T) {
	s := NewStore()
	g := s.SaveGrid(newTestGrid(10, 10))

	if g.ID == "" {
		t.Fatal("expected grid to have an ID")
	}
	if got := s.GetGrid(g.ID); got == nil {
		t.Fatal("expected to find saved grid")
	}
	if got := s.GetGrid("nonexistent"); got != nil {
		t.Fatal("expected nil for unknown ID")
	}
}

func TestListGrids(t *testing.T) {
	s := NewStore()
	s.SaveGrid(newTestGrid(5, 5))
	s.SaveGrid(newTestGrid(8, 8))

	list := s.ListGrids()
	if len(list) != 2 {
		t.Fatalf("expected 2 grids, got %d", len(list))
	}
	// Most recent first.
	if list[0].CreatedAt.Before(list[1].CreatedAt) {
		t.Fatal("expected grids sorted by descending creation time")
	}
}

func TestCreateGame(t *testing.T) {
	s := NewStore()

	// Error on unknown grid.
	if _, err := s.CreateGame("unknown"); err == nil {
		t.Fatal("expected error for unknown grid")
	}

	g := s.SaveGrid(newTestGrid(3, 4))
	game, err := s.CreateGame(g.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if game.GridID != g.ID {
		t.Fatal("game should reference the grid")
	}
	if len(game.State) != 3 || len(game.State[0]) != 4 {
		t.Fatalf("expected 3x4 state, got %dx%d", len(game.State), len(game.State[0]))
	}
}

func TestGameAddPlayer(t *testing.T) {
	s := NewStore()
	g := s.SaveGrid(newTestGrid(5, 5))
	game, _ := s.CreateGame(g.ID)

	p1 := game.AddPlayer("Alice")
	p2 := game.AddPlayer("Bob")

	if p1.Pseudo != "Alice" || p2.Pseudo != "Bob" {
		t.Fatal("unexpected pseudo")
	}
	if p1.Color == p2.Color {
		t.Fatal("players should have different colors")
	}

	// Adding same pseudo returns existing player.
	p1bis := game.AddPlayer("Alice")
	if p1bis.Color != p1.Color {
		t.Fatal("same pseudo should return same player")
	}
}

func TestGameSetCell(t *testing.T) {
	s := NewStore()
	g := s.SaveGrid(newTestGrid(3, 3))
	game, _ := s.CreateGame(g.ID)

	if !game.SetCell(0, 0, "A") {
		t.Fatal("expected SetCell to succeed")
	}
	if game.SetCell(-1, 0, "X") {
		t.Fatal("expected SetCell to fail for negative row")
	}
	if game.SetCell(0, 3, "X") {
		t.Fatal("expected SetCell to fail for out-of-bounds col")
	}

	state := game.GetState()
	if state[0][0] != "A" {
		t.Fatalf("expected 'A', got %q", state[0][0])
	}
}

func TestGetStateCopy(t *testing.T) {
	s := NewStore()
	g := s.SaveGrid(newTestGrid(2, 2))
	game, _ := s.CreateGame(g.ID)
	game.SetCell(0, 0, "X")

	state := game.GetState()
	state[0][0] = "Z" // mutate the copy

	original := game.GetState()
	if original[0][0] != "X" {
		t.Fatal("GetState should return a copy, not a reference")
	}
}

func TestConcurrentAccess(t *testing.T) {
	s := NewStore()
	g := s.SaveGrid(newTestGrid(10, 10))
	game, _ := s.CreateGame(g.ID)

	var wg sync.WaitGroup
	for i := range 100 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			game.SetCell(i%10, i%10, "A")
			game.GetState()
			game.AddPlayer("player" + string(rune('A'+i%26)))
		}(i)
	}
	wg.Wait()
}
