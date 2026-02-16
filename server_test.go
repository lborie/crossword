package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newTestServer() *Server {
	store := NewStore()
	return NewServer(store, nil)
}

func seedGrid(s *Server) *Grid {
	g := &Grid{
		Rows: 3,
		Cols: 3,
		Cells: [][]Cell{
			{{Black: true, Definitions: []Definition{{Text: "Test", Direction: "right"}}}, {}, {}},
			{{Black: true, Definitions: []Definition{{Text: "Down", Direction: "down"}}}, {}, {}},
			{{}, {}, {}},
		},
	}
	s.store.SaveGrid(g)
	return g
}

func TestGamePageRoute(t *testing.T) {
	srv := newTestServer()

	req := httptest.NewRequest("GET", "/game/abc123", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Fatalf("expected text/html, got %s", ct)
	}
	if !strings.Contains(w.Body.String(), "Mots Croisés") {
		t.Fatal("game page does not contain expected title")
	}
}

func TestFullGameFlow(t *testing.T) {
	srv := newTestServer()
	grid := seedGrid(srv)

	// Create game.
	body := `{"grid_id":"` + grid.ID + `"}`
	req := httptest.NewRequest("POST", "/api/games", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("create game: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var game GameSession
	json.NewDecoder(w.Body).Decode(&game)
	if game.ID == "" {
		t.Fatal("game ID is empty")
	}

	// Join game.
	body = `{"pseudo":"Alice"}`
	req = httptest.NewRequest("POST", "/api/games/"+game.ID+"/join", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("join game: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var player Player
	json.NewDecoder(w.Body).Decode(&player)
	if player.Pseudo != "Alice" {
		t.Fatalf("expected pseudo Alice, got %s", player.Pseudo)
	}

	// Place a letter on a valid cell (row=0, col=1).
	body = `{"pseudo":"Alice","row":0,"col":1,"value":"A"}`
	req = httptest.NewRequest("POST", "/api/games/"+game.ID+"/move", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("move: expected 204, got %d: %s", w.Code, w.Body.String())
	}

	// Try to place on a definition cell (row=0, col=0).
	body = `{"pseudo":"Alice","row":0,"col":0,"value":"B"}`
	req = httptest.NewRequest("POST", "/api/games/"+game.ID+"/move", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("move on def cell: expected 400, got %d", w.Code)
	}

	// Get game state — verify the letter is there.
	req = httptest.NewRequest("GET", "/api/games/"+game.ID, nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("get game: expected 200, got %d", w.Code)
	}

	var resp struct {
		State [][]string `json:"state"`
		Grid  *Grid      `json:"grid"`
	}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.State[0][1] != "A" {
		t.Fatalf("expected cell (0,1) = 'A', got %q", resp.State[0][1])
	}
	if resp.Grid == nil {
		t.Fatal("grid should be included in game response")
	}
}

func TestCreateGameInvalidGrid(t *testing.T) {
	srv := newTestServer()

	body := `{"grid_id":"nonexistent"}`
	req := httptest.NewRequest("POST", "/api/games", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestMoveValidation(t *testing.T) {
	srv := newTestServer()
	grid := seedGrid(srv)

	// Create game.
	body := `{"grid_id":"` + grid.ID + `"}`
	req := httptest.NewRequest("POST", "/api/games", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	var game GameSession
	json.NewDecoder(w.Body).Decode(&game)

	// Invalid value (number).
	body = `{"pseudo":"Bob","row":0,"col":1,"value":"5"}`
	req = httptest.NewRequest("POST", "/api/games/"+game.ID+"/move", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid value, got %d", w.Code)
	}

	// Out of bounds.
	body = `{"pseudo":"Bob","row":10,"col":10,"value":"A"}`
	req = httptest.NewRequest("POST", "/api/games/"+game.ID+"/move", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for out of bounds, got %d", w.Code)
	}

	// Empty value (erase) — should succeed.
	body = `{"pseudo":"Bob","row":2,"col":2,"value":""}`
	req = httptest.NewRequest("POST", "/api/games/"+game.ID+"/move", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for erase, got %d", w.Code)
	}
}

func TestSecurityHeaders(t *testing.T) {
	srv := newTestServer()

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	headers := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":       "DENY",
		"Referrer-Policy":       "strict-origin-when-cross-origin",
	}

	for key, expected := range headers {
		if got := w.Header().Get(key); got != expected {
			t.Errorf("header %s: expected %q, got %q", key, expected, got)
		}
	}

	csp := w.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Error("Content-Security-Policy header missing")
	}
}

func TestRateLimiter(t *testing.T) {
	rl := newRateLimiter(3, time.Second)

	// First 3 should pass.
	for i := range 3 {
		if !rl.allow("1.2.3.4") {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}

	// 4th should be blocked.
	if rl.allow("1.2.3.4") {
		t.Fatal("4th request should be rate limited")
	}

	// Different IP should still be allowed.
	if !rl.allow("5.6.7.8") {
		t.Fatal("different IP should be allowed")
	}
}
