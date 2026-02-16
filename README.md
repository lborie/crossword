# Crossword

Mots croisés collaboratifs en temps réel.

## Principe

1. Prenez en photo une grille de mots croisés (mots flechs)
2. L'application analyse la photo via Gemini Flash (VertexAI) et reconstruit la grille
3. Creez une partie et partagez le lien
4. Completez la grille a plusieurs, en temps reel

## Stack technique

| Composant | Technologie |
|-----------|-------------|
| Backend | Go (net/http, embed) |
| Frontend | HTML / CSS / JS vanilla |
| Vision IA | Gemini 2.5 Flash (VertexAI) |
| Temps reel | Server-Sent Events (SSE) |
| Auth GCP | google.golang.org/genai SDK |

## Demarrage

```bash
# Variables d'environnement
export GCP_PROJECT_ID=votre-projet
export GCP_REGION=europe-west1              # optionnel, defaut: europe-west1
export GOOGLE_APPLICATION_CREDENTIALS=chemin/vers/credentials.json

# Lancer le serveur
go run .
# -> http://localhost:8080
```

Sans `GCP_PROJECT_ID`, le serveur demarre mais l'upload de grilles est desactive.

## API

| Methode | Route | Description |
|---------|-------|-------------|
| `POST /api/grids` | multipart (image) | Upload photo, analyse Gemini, cree grille |
| `GET /api/grids` | | Liste des grilles |
| `GET /api/grids/{id}` | | Detail d'une grille |
| `POST /api/games` | `{grid_id}` | Creer une partie |
| `GET /api/games/{id}` | | Etat d'une partie (avec grille) |
| `POST /api/games/{id}/join` | `{pseudo}` | Rejoindre une partie |
| `POST /api/games/{id}/move` | `{pseudo, row, col, value}` | Poser/effacer une lettre |
| `GET /api/games/{id}/events` | SSE | Flux temps reel |

## Fonctionnalites

- Analyse d'image par IA (extraction grille + definitions + directions)
- Grille interactive avec navigation clavier (fleches, Tab, Backspace)
- Mise en surbrillance du mot en cours
- Affichage de la definition courante
- Synchronisation temps reel entre joueurs (SSE)
- Reconnexion automatique avec backoff exponentiel
- Liste des joueurs avec couleurs
- Notification d'arrivee/depart des joueurs
- Responsive mobile-first
- Headers de securite (CSP, X-Frame-Options, X-Content-Type-Options)
- Rate limiting sur upload et moves
