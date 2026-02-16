package main

import (
	"embed"
	"encoding/json"
	"io"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

//go:embed frontend
var frontendFS embed.FS

const maxUploadSize = 10 << 20 // 10 Mo

var allowedMIME = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
}

// rateLimiter is a simple per-IP token bucket rate limiter.
type rateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*bucket
	rate     int           // tokens per interval
	interval time.Duration // refill interval
}

type bucket struct {
	tokens   int
	lastSeen time.Time
}

func newRateLimiter(rate int, interval time.Duration) *rateLimiter {
	rl := &rateLimiter{
		visitors: make(map[string]*bucket),
		rate:     rate,
		interval: interval,
	}
	// Cleanup stale entries every minute.
	go func() {
		for {
			time.Sleep(time.Minute)
			rl.mu.Lock()
			for ip, b := range rl.visitors {
				if time.Since(b.lastSeen) > 5*time.Minute {
					delete(rl.visitors, ip)
				}
			}
			rl.mu.Unlock()
		}
	}()
	return rl
}

func (rl *rateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	b, ok := rl.visitors[ip]
	if !ok {
		rl.visitors[ip] = &bucket{tokens: rl.rate - 1, lastSeen: time.Now()}
		return true
	}

	// Refill tokens based on elapsed time.
	elapsed := time.Since(b.lastSeen)
	refill := int(elapsed / rl.interval)
	if refill > 0 {
		b.tokens += refill * rl.rate
		if b.tokens > rl.rate {
			b.tokens = rl.rate
		}
		b.lastSeen = time.Now()
	}

	if b.tokens <= 0 {
		return false
	}
	b.tokens--
	return true
}

// Server is the main HTTP server.
type Server struct {
	mux        *http.ServeMux
	store      *Store
	gemini     *GeminiClient
	sse        *Broadcaster
	uploadRL   *rateLimiter
	moveRL     *rateLimiter
}

// NewServer creates a configured HTTP server.
func NewServer(store *Store, gemini *GeminiClient) *Server {
	s := &Server{
		mux:      http.NewServeMux(),
		store:    store,
		gemini:   gemini,
		sse:      NewBroadcaster(),
		uploadRL: newRateLimiter(5, time.Minute),   // 5 uploads/min per IP
		moveRL:   newRateLimiter(60, time.Second),   // 60 moves/sec per IP
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	// Grid API
	s.mux.HandleFunc("POST /api/grids", s.handleCreateGrid)
	s.mux.HandleFunc("GET /api/grids", s.handleListGrids)
	s.mux.HandleFunc("GET /api/grids/{id}", s.handleGetGrid)

	// Game API
	s.mux.HandleFunc("POST /api/games", s.handleCreateGame)
	s.mux.HandleFunc("GET /api/games/{id}", s.handleGetGame)
	s.mux.HandleFunc("POST /api/games/{id}/join", s.handleJoinGame)
	s.mux.HandleFunc("POST /api/games/{id}/move", s.handleMove)
	s.mux.HandleFunc("GET /api/games/{id}/events", s.handleGameEvents)

	// Frontend static files
	frontendDir, _ := fs.Sub(frontendFS, "frontend")
	fileServer := http.FileServer(http.FS(frontendDir))
	s.mux.HandleFunc("GET /game/{id}", s.handleGamePage)
	s.mux.Handle("GET /", fileServer)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'")
	s.mux.ServeHTTP(w, r)
}

// --- Grid handlers ---

// POST /api/grids — upload image, analyze with Gemini, save grid.
func (s *Server) handleCreateGrid(w http.ResponseWriter, r *http.Request) {
	if !s.uploadRL.allow(r.RemoteAddr) {
		jsonError(w, "Trop de requêtes, réessayez plus tard", http.StatusTooManyRequests)
		return
	}

	if s.gemini == nil {
		jsonError(w, "Analyse d'image non configurée", http.StatusServiceUnavailable)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		jsonError(w, "Image trop volumineuse (max 10 Mo)", http.StatusRequestEntityTooLarge)
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		jsonError(w, "Champ 'image' requis", http.StatusBadRequest)
		return
	}
	defer file.Close()

	mimeType := header.Header.Get("Content-Type")
	if !allowedMIME[mimeType] {
		jsonError(w, "Format accepté : JPEG ou PNG", http.StatusBadRequest)
		return
	}

	imageData, err := io.ReadAll(file)
	if err != nil {
		jsonError(w, "Erreur de lecture de l'image", http.StatusInternalServerError)
		return
	}

	grid, err := s.gemini.AnalyzeImage(r.Context(), imageData, mimeType)
	if err != nil {
		log.Printf("Gemini analyze error: %v", err)
		jsonError(w, "Erreur lors de l'analyse de la grille", http.StatusInternalServerError)
		return
	}

	s.store.SaveGrid(grid)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(grid)
}

// GET /api/grids — list all grids.
func (s *Server) handleListGrids(w http.ResponseWriter, _ *http.Request) {
	grids := s.store.ListGrids()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(grids)
}

// GET /api/grids/{id} — get a single grid.
func (s *Server) handleGetGrid(w http.ResponseWriter, r *http.Request) {
	grid := s.store.GetGrid(r.PathValue("id"))
	if grid == nil {
		jsonError(w, "Grille introuvable", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(grid)
}

// --- Game handlers ---

// POST /api/games — create a game from a grid.
func (s *Server) handleCreateGame(w http.ResponseWriter, r *http.Request) {
	var req struct {
		GridID string `json:"grid_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.GridID == "" {
		jsonError(w, "Champ 'grid_id' requis", http.StatusBadRequest)
		return
	}

	game, err := s.store.CreateGame(req.GridID)
	if err != nil {
		jsonError(w, "Grille introuvable", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(game)
}

// GET /api/games/{id} — get current game state.
func (s *Server) handleGetGame(w http.ResponseWriter, r *http.Request) {
	game := s.store.GetGame(r.PathValue("id"))
	if game == nil {
		jsonError(w, "Partie introuvable", http.StatusNotFound)
		return
	}

	resp := struct {
		*GameSession
		Grid *Grid `json:"grid"`
	}{
		GameSession: game,
		Grid:        s.store.GetGrid(game.GridID),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// POST /api/games/{id}/join — join a game with a pseudo.
func (s *Server) handleJoinGame(w http.ResponseWriter, r *http.Request) {
	game := s.store.GetGame(r.PathValue("id"))
	if game == nil {
		jsonError(w, "Partie introuvable", http.StatusNotFound)
		return
	}

	var req struct {
		Pseudo string `json:"pseudo"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Pseudo == "" {
		jsonError(w, "Champ 'pseudo' requis", http.StatusBadRequest)
		return
	}

	pseudo := sanitizePseudo(req.Pseudo)
	if pseudo == "" {
		jsonError(w, "Pseudo invalide", http.StatusBadRequest)
		return
	}

	player := game.AddPlayer(pseudo)

	// Broadcast player_joined event.
	evt, _ := json.Marshal(map[string]string{
		"type":   "player_joined",
		"pseudo": player.Pseudo,
		"color":  player.Color,
	})
	s.sse.Broadcast(game.ID, string(evt))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(player)
}

// POST /api/games/{id}/move — place a letter.
func (s *Server) handleMove(w http.ResponseWriter, r *http.Request) {
	if !s.moveRL.allow(r.RemoteAddr) {
		jsonError(w, "Trop de requêtes, réessayez plus tard", http.StatusTooManyRequests)
		return
	}

	game := s.store.GetGame(r.PathValue("id"))
	if game == nil {
		jsonError(w, "Partie introuvable", http.StatusNotFound)
		return
	}

	var req struct {
		Pseudo string `json:"pseudo"`
		Row    int    `json:"row"`
		Col    int    `json:"col"`
		Value  string `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Requête invalide", http.StatusBadRequest)
		return
	}

	// Validate: value must be empty (erase) or a single uppercase letter.
	value := strings.ToUpper(strings.TrimSpace(req.Value))
	if value != "" && (utf8.RuneCountInString(value) != 1 || value < "A" || value > "Z") {
		jsonError(w, "Valeur invalide : une lettre A-Z ou vide", http.StatusBadRequest)
		return
	}

	// Check the cell is not a definition cell.
	grid := s.store.GetGrid(game.GridID)
	if grid != nil && req.Row >= 0 && req.Row < grid.Rows && req.Col >= 0 && req.Col < grid.Cols {
		if grid.Cells[req.Row][req.Col].Black {
			jsonError(w, "Case de définition", http.StatusBadRequest)
			return
		}
	}

	if !game.SetCell(req.Row, req.Col, value) {
		jsonError(w, "Position hors limites", http.StatusBadRequest)
		return
	}

	// Broadcast cell_update event.
	evt, _ := json.Marshal(map[string]any{
		"type":   "cell_update",
		"row":    req.Row,
		"col":    req.Col,
		"value":  value,
		"pseudo": req.Pseudo,
	})
	s.sse.Broadcast(game.ID, string(evt))

	w.WriteHeader(http.StatusNoContent)
}

// GET /api/games/{id}/events — SSE stream.
func (s *Server) handleGameEvents(w http.ResponseWriter, r *http.Request) {
	game := s.store.GetGame(r.PathValue("id"))
	if game == nil {
		jsonError(w, "Partie introuvable", http.StatusNotFound)
		return
	}

	playerPseudo := sanitizePseudo(r.URL.Query().Get("pseudo"))

	s.sse.ServeSSE(w, r, game.ID, func(c *client) {
		// Send initial game state on connect.
		evt, _ := json.Marshal(map[string]any{
			"type":    "game_state",
			"state":   game.GetState(),
			"players": game.Players,
		})
		c.ch <- string(evt)
	}, func() {
		// On disconnect: broadcast player_left if pseudo was provided.
		if playerPseudo != "" {
			game.RemovePlayer(playerPseudo)
			evt, _ := json.Marshal(map[string]string{
				"type":   "player_left",
				"pseudo": playerPseudo,
			})
			s.sse.Broadcast(game.ID, string(evt))
		}
	})
}

// --- Frontend page handlers ---

// GET /game/{id} — serve the game page.
func (s *Server) handleGamePage(w http.ResponseWriter, _ *http.Request) {
	data, _ := frontendFS.ReadFile("frontend/game.html")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}

// --- Helpers ---

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func sanitizePseudo(s string) string {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) > 20 {
		s = string([]rune(s)[:20])
	}
	return s
}
