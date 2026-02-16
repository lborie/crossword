package main

import (
	"context"
	"encoding/json"
	"fmt"

	"google.golang.org/genai"
)

const analyzePrompt = `Analyse cette photo de grille de mots fléchés.

Extrais la structure complète au format JSON suivant :
{
  "rows": <nombre de lignes>,
  "cols": <nombre de colonnes>,
  "cells": [
    [
      {"black": true, "definitions": [{"text": "Définition", "direction": "right"}]},
      {"black": false},
      ...
    ],
    ...
  ]
}

Règles :
- Chaque case contenant du texte et/ou une flèche est une case définition : "black": true avec "definitions".
- "direction" vaut "right" si la flèche pointe vers la droite, "down" si elle pointe vers le bas.
- Une case définition peut avoir 1 ou 2 définitions (une vers la droite, une vers le bas).
- Les cases vides (où le joueur écrit) ont "black": false et pas de "definitions".
- Réponds UNIQUEMENT avec le JSON, sans commentaire ni markdown.`

// AnalyzeImage sends an image to Gemini Flash and returns the extracted grid.
func (g *GeminiClient) AnalyzeImage(ctx context.Context, imageData []byte, mimeType string) (*Grid, error) {
	resp, err := g.client.Models.GenerateContent(ctx, g.modelName,
		[]*genai.Content{{
			Role: "user",
			Parts: []*genai.Part{
				{Text: analyzePrompt},
				{InlineData: &genai.Blob{MIMEType: mimeType, Data: imageData}},
			},
		}},
		&genai.GenerateContentConfig{
			Temperature:      genai.Ptr(float32(0.1)),
			TopP:             genai.Ptr(float32(1)),
			ResponseMIMEType: "application/json",
		},
	)
	if err != nil {
		return nil, fmt.Errorf("gemini generate: %w", err)
	}

	text := resp.Text()
	if text == "" {
		return nil, fmt.Errorf("empty gemini response")
	}

	var grid Grid
	if err := json.Unmarshal([]byte(text), &grid); err != nil {
		return nil, fmt.Errorf("parse grid JSON: %w\nraw response: %s", err, text)
	}

	if grid.Rows == 0 || grid.Cols == 0 || len(grid.Cells) == 0 {
		return nil, fmt.Errorf("invalid grid: %dx%d with %d cell rows", grid.Rows, grid.Cols, len(grid.Cells))
	}

	return &grid, nil
}
