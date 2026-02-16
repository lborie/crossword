package main

import (
	"context"
	"encoding/json"
	"os"
	"testing"
)

func TestAnalyzeImage(t *testing.T) {
	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		t.Skip("GCP_PROJECT_ID not set, skipping integration test")
	}

	ctx := context.Background()
	client, err := NewGeminiClient(ctx, projectID, "")
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	defer client.Close()

	imageData, err := os.ReadFile("test_data/example.png")
	if err != nil {
		t.Fatalf("read image: %v", err)
	}

	grid, err := client.AnalyzeImage(ctx, imageData, "image/png")
	if err != nil {
		t.Fatalf("analyze image: %v", err)
	}

	if grid.Rows == 0 || grid.Cols == 0 {
		t.Fatalf("invalid dimensions: %dx%d", grid.Rows, grid.Cols)
	}
	if len(grid.Cells) != grid.Rows {
		t.Fatalf("expected %d cell rows, got %d", grid.Rows, len(grid.Cells))
	}

	// Count definition cells.
	defCount := 0
	for _, row := range grid.Cells {
		for _, cell := range row {
			if cell.Black {
				defCount++
			}
		}
	}

	t.Logf("Grid: %dx%d, %d definition cells", grid.Rows, grid.Cols, defCount)

	// Print a sample for manual inspection.
	out, _ := json.MarshalIndent(grid, "", "  ")
	t.Logf("Extracted grid:\n%s", string(out))
}
