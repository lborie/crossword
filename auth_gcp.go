package main

import (
	"context"
	"fmt"

	"google.golang.org/genai"
)

const (
	defaultRegion = "europe-west1"
	defaultModel  = "gemini-2.5-flash"
)

// GeminiClient wraps the Google GenAI client for VertexAI.
type GeminiClient struct {
	client    *genai.Client
	modelName string
}

// NewGeminiClient creates a client using Application Default Credentials.
// Set GOOGLE_APPLICATION_CREDENTIALS to the service account key file path.
func NewGeminiClient(ctx context.Context, projectID, region string) (*GeminiClient, error) {
	if region == "" {
		region = defaultRegion
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Project:  projectID,
		Location: region,
		Backend:  genai.BackendVertexAI,
	})
	if err != nil {
		return nil, fmt.Errorf("create genai client: %w", err)
	}

	return &GeminiClient{
		client:    client,
		modelName: defaultModel,
	}, nil
}

// Close releases resources held by the client.
func (g *GeminiClient) Close() error {
	return nil
}
