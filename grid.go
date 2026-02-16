package main

import "time"

// Definition is a clue embedded in a definition cell (mots fléchés).
type Definition struct {
	Text      string `json:"text"`
	Direction string `json:"direction"` // "right" or "down"
}

// Cell represents a single cell in the crossword grid.
// A cell is either a definition cell (Black=true, with Definitions)
// or a letter cell (Black=false, where players write).
type Cell struct {
	Black       bool         `json:"black"`
	Definitions []Definition `json:"definitions,omitempty"`
}

// Grid represents a crossword grid extracted from an image.
type Grid struct {
	ID        string     `json:"id"`
	Rows      int        `json:"rows"`
	Cols      int        `json:"cols"`
	Cells     [][]Cell   `json:"cells"`
	CreatedAt time.Time  `json:"created_at"`
}
