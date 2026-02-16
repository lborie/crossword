package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

const (
	sseChannelBuffer = 16
	sseHeartbeat     = 30 * time.Second
)

// client represents a single SSE connection.
type client struct {
	ch     chan string
	gameID string
}

// Broadcaster manages SSE clients grouped by game session.
type Broadcaster struct {
	mu      sync.RWMutex
	clients map[*client]struct{}
}

// NewBroadcaster creates an empty broadcaster.
func NewBroadcaster() *Broadcaster {
	return &Broadcaster{
		clients: make(map[*client]struct{}),
	}
}

// Register adds a client for a game session and returns it.
func (b *Broadcaster) Register(gameID string) *client {
	c := &client{
		ch:     make(chan string, sseChannelBuffer),
		gameID: gameID,
	}
	b.mu.Lock()
	b.clients[c] = struct{}{}
	b.mu.Unlock()
	return c
}

// Unregister removes a client and closes its channel.
func (b *Broadcaster) Unregister(c *client) {
	b.mu.Lock()
	if _, ok := b.clients[c]; ok {
		delete(b.clients, c)
		close(c.ch)
	}
	b.mu.Unlock()
}

// Broadcast sends a message to all clients of a game session.
func (b *Broadcaster) Broadcast(gameID, data string) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for c := range b.clients {
		if c.gameID == gameID {
			select {
			case c.ch <- data:
			default:
				// Channel full, skip slow client.
			}
		}
	}
}

// ClientCount returns the number of connected clients for a game.
func (b *Broadcaster) ClientCount(gameID string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	n := 0
	for c := range b.clients {
		if c.gameID == gameID {
			n++
		}
	}
	return n
}

// ServeSSE handles an SSE connection for a game session.
func (b *Broadcaster) ServeSSE(w http.ResponseWriter, r *http.Request, gameID string, onConnect func(c *client), onDisconnect func()) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming non supporté", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	c := b.Register(gameID)
	defer func() {
		b.Unregister(c)
		if onDisconnect != nil {
			onDisconnect()
		}
	}()

	if onConnect != nil {
		onConnect(c)
	}

	ticker := time.NewTicker(sseHeartbeat)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case msg, ok := <-c.ch:
			if !ok {
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		case <-ticker.C:
			fmt.Fprintf(w, ": heartbeat\n\n")
			flusher.Flush()
		}
	}
}
