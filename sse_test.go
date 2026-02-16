package main

import (
	"sync"
	"testing"
	"time"
)

func TestBroadcasterRegisterUnregister(t *testing.T) {
	b := NewBroadcaster()

	c1 := b.Register("game1")
	c2 := b.Register("game1")
	c3 := b.Register("game2")

	if b.ClientCount("game1") != 2 {
		t.Fatalf("expected 2 clients for game1, got %d", b.ClientCount("game1"))
	}
	if b.ClientCount("game2") != 1 {
		t.Fatalf("expected 1 client for game2, got %d", b.ClientCount("game2"))
	}

	b.Unregister(c1)
	if b.ClientCount("game1") != 1 {
		t.Fatalf("expected 1 client for game1 after unregister, got %d", b.ClientCount("game1"))
	}

	b.Unregister(c2)
	b.Unregister(c3)
	if b.ClientCount("game1") != 0 || b.ClientCount("game2") != 0 {
		t.Fatal("expected 0 clients after full unregister")
	}
}

func TestBroadcasterDoubleUnregister(t *testing.T) {
	b := NewBroadcaster()
	c := b.Register("game1")
	b.Unregister(c)
	b.Unregister(c) // should not panic
}

func TestBroadcast(t *testing.T) {
	b := NewBroadcaster()

	c1 := b.Register("game1")
	c2 := b.Register("game1")
	c3 := b.Register("game2")

	b.Broadcast("game1", "hello")

	select {
	case msg := <-c1.ch:
		if msg != "hello" {
			t.Fatalf("c1 expected 'hello', got %q", msg)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("c1 did not receive message")
	}

	select {
	case msg := <-c2.ch:
		if msg != "hello" {
			t.Fatalf("c2 expected 'hello', got %q", msg)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("c2 did not receive message")
	}

	// c3 is on game2, should not receive.
	select {
	case <-c3.ch:
		t.Fatal("c3 should not receive game1 message")
	case <-time.After(50 * time.Millisecond):
		// ok
	}

	b.Unregister(c1)
	b.Unregister(c2)
	b.Unregister(c3)
}

func TestBroadcastSkipsFullChannel(t *testing.T) {
	b := NewBroadcaster()
	c := b.Register("game1")

	// Fill the channel.
	for range sseChannelBuffer {
		b.Broadcast("game1", "fill")
	}

	// This should not block.
	b.Broadcast("game1", "overflow")

	b.Unregister(c)
}

func TestBroadcasterConcurrent(t *testing.T) {
	b := NewBroadcaster()
	var wg sync.WaitGroup

	for i := range 50 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			gameID := "game1"
			if i%2 == 0 {
				gameID = "game2"
			}
			c := b.Register(gameID)
			b.Broadcast(gameID, "msg")
			b.ClientCount(gameID)
			b.Unregister(c)
		}(i)
	}
	wg.Wait()

	if b.ClientCount("game1") != 0 || b.ClientCount("game2") != 0 {
		t.Fatal("expected 0 clients after concurrent test")
	}
}
