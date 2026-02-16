package main

import (
	"context"
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	ctx := context.Background()

	projectID := os.Getenv("GCP_PROJECT_ID")

	var gemini *GeminiClient
	if projectID != "" {
		var err error
		gemini, err = NewGeminiClient(ctx, projectID, os.Getenv("GCP_REGION"))
		if err != nil {
			log.Fatalf("Impossible d'initialiser Gemini : %v", err)
		}
		defer gemini.Close()
		log.Printf("Client Gemini initialisé (projet: %s)", projectID)
	} else {
		log.Println("GCP_PROJECT_ID non défini — analyse d'image désactivée")
	}

	srv := NewServer(NewStore(), gemini)

	log.Printf("Serveur démarré sur http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, srv); err != nil {
		log.Fatal(err)
	}
}
