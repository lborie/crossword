# Crossword

Mots croisés collaboratifs en temps réel.

## Principe

1. Prenez en photo une grille de mots croisés
2. L'application analyse la photo via Gemini Flash (VertexAI) et reconstruit la grille
3. Partagez la grille et complétez-la à plusieurs, en temps réel

## Stack technique

| Composant | Technologie |
|-----------|-------------|
| Backend | Go |
| Frontend | HTML / CSS / JS vanilla |
| Vision IA | Gemini Flash (VertexAI) |
| Temps réel | Server-Sent Events (SSE) |

Aucune dépendance externe, ni côté serveur ni côté client.
