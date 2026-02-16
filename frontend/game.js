"use strict";

const $ = (sel) => document.querySelector(sel);

// Extract game ID from URL: /game/{id}
const gameID = location.pathname.split("/").pop();

let grid = null;       // Grid data (cells, rows, cols)
let state = null;      // Current game state [row][col]
let pseudo = null;     // Current player pseudo
let eventSource = null;
let selectedRow = -1;
let selectedCol = -1;
let direction = "right"; // "right" or "down"

// --- Join ---

const joinSection = $("#join-section");
const joinForm = $("#join-form");
const pseudoInput = $("#pseudo-input");
const gameArea = $("#game-area");

joinForm.addEventListener("submit", async (e) => {
    e.preventDefault();
    const name = pseudoInput.value.trim();
    if (!name) return;

    try {
        const resp = await fetch("/api/games/" + encodeURIComponent(gameID) + "/join", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ pseudo: name }),
        });
        if (!resp.ok) {
            const data = await resp.json();
            throw new Error(data.error || "Erreur");
        }
        pseudo = name;
        joinSection.hidden = true;
        gameArea.hidden = false;
        loadGame();
    } catch (err) {
        showJoinError(err.message);
    }
});

// --- Load game ---

async function loadGame() {
    try {
        const resp = await fetch("/api/games/" + encodeURIComponent(gameID));
        if (!resp.ok) throw new Error("Partie introuvable");
        const data = await resp.json();
        grid = data.grid;
        state = data.state;
        renderPlayers(data.players);
        renderGrid();
        connectSSE();
    } catch (err) {
        showJoinError(err.message);
    }
}

// --- Render grid ---

function renderGrid() {
    const table = $("#game-grid");
    table.textContent = "";

    for (let r = 0; r < grid.rows; r++) {
        const tr = document.createElement("tr");
        for (let c = 0; c < grid.cols; c++) {
            const cell = grid.cells[r][c];
            const td = document.createElement("td");
            td.dataset.row = r;
            td.dataset.col = c;

            if (cell.black) {
                td.className = "cell-def";
                renderDefsInto(td, cell.definitions);
            } else {
                td.className = "cell-letter";
                td.tabIndex = 0;
                if (state[r][c]) {
                    td.textContent = state[r][c];
                }
                td.addEventListener("click", () => selectCell(r, c));
            }
            tr.appendChild(td);
        }
        table.appendChild(tr);
    }

    // Select first available cell.
    selectFirstCell();
}

function renderDefsInto(td, defs) {
    if (!defs || defs.length === 0) return;
    for (const d of defs) {
        const text = document.createElement("span");
        text.className = "def-text";
        text.textContent = d.text;

        const arrow = document.createElement("span");
        arrow.className = "def-arrow";
        arrow.textContent = d.direction === "right" ? "\u2192" : "\u2193";

        td.appendChild(text);
        td.appendChild(arrow);
    }
}

function selectFirstCell() {
    for (let r = 0; r < grid.rows; r++) {
        for (let c = 0; c < grid.cols; c++) {
            if (!grid.cells[r][c].black) {
                selectCell(r, c);
                return;
            }
        }
    }
}

function selectCell(row, col) {
    // If clicking the same cell, toggle direction.
    if (row === selectedRow && col === selectedCol) {
        direction = direction === "right" ? "down" : "right";
    }

    selectedRow = row;
    selectedCol = col;

    // Update visual selection.
    const table = $("#game-grid");
    const cells = table.querySelectorAll("td");
    for (const td of cells) {
        td.classList.remove("selected", "highlighted");
    }

    const selected = getCell(row, col);
    if (selected) {
        selected.classList.add("selected");
        selected.focus();
    }

    // Highlight the word in current direction.
    highlightWord(row, col);

    // Show current definition.
    showDefinition(row, col);
}

function getCell(row, col) {
    return $("#game-grid").querySelector(
        'td[data-row="' + row + '"][data-col="' + col + '"]'
    );
}

function highlightWord(row, col) {
    if (direction === "right") {
        // Highlight all letter cells in the same row, from the nearest def cell to the left.
        let startCol = col;
        while (startCol > 0 && !grid.cells[row][startCol - 1].black) {
            startCol--;
        }
        for (let c = startCol; c < grid.cols && !grid.cells[row][c].black; c++) {
            const td = getCell(row, c);
            if (td && c !== col) td.classList.add("highlighted");
        }
    } else {
        // Highlight all letter cells in the same column, from the nearest def cell above.
        let startRow = row;
        while (startRow > 0 && !grid.cells[startRow - 1][col].black) {
            startRow--;
        }
        for (let r = startRow; r < grid.rows && !grid.cells[r][col].black; r++) {
            const td = getCell(r, col);
            if (td && r !== row) td.classList.add("highlighted");
        }
    }
}

function showDefinition(row, col) {
    const defSection = $("#current-def");
    const defText = $("#def-text");

    let def = null;

    if (direction === "right") {
        // Find the definition cell to the left of this word.
        let c = col;
        while (c > 0 && !grid.cells[row][c - 1].black) {
            c--;
        }
        // Check cell at c-1 (or c if it's at the edge).
        if (c > 0) {
            const defCell = grid.cells[row][c - 1];
            if (defCell.definitions) {
                def = defCell.definitions.find((d) => d.direction === "right");
            }
        }
    } else {
        // Find the definition cell above this word.
        let r = row;
        while (r > 0 && !grid.cells[r - 1][col].black) {
            r--;
        }
        if (r > 0) {
            const defCell = grid.cells[r - 1][col];
            if (defCell.definitions) {
                def = defCell.definitions.find((d) => d.direction === "down");
            }
        }
    }

    if (def) {
        defText.textContent = (direction === "right" ? "\u2192 " : "\u2193 ") + def.text;
        defSection.hidden = false;
    } else {
        defSection.hidden = true;
    }
}

// --- Keyboard ---

document.addEventListener("keydown", (e) => {
    if (selectedRow < 0 || selectedCol < 0) return;
    if (joinSection && !joinSection.hidden) return;

    if (e.key === "ArrowRight") {
        e.preventDefault();
        moveSelection(0, 1);
    } else if (e.key === "ArrowLeft") {
        e.preventDefault();
        moveSelection(0, -1);
    } else if (e.key === "ArrowDown") {
        e.preventDefault();
        moveSelection(1, 0);
    } else if (e.key === "ArrowUp") {
        e.preventDefault();
        moveSelection(-1, 0);
    } else if (e.key === "Tab") {
        e.preventDefault();
        direction = direction === "right" ? "down" : "right";
        selectCell(selectedRow, selectedCol);
    } else if (e.key === "Backspace" || e.key === "Delete") {
        e.preventDefault();
        sendMove(selectedRow, selectedCol, "");
        if (e.key === "Backspace") {
            movePrev();
        }
    } else if (/^[a-zA-Z]$/.test(e.key)) {
        e.preventDefault();
        sendMove(selectedRow, selectedCol, e.key.toUpperCase());
        moveNext();
    }
});

function moveSelection(dRow, dCol) {
    let r = selectedRow + dRow;
    let c = selectedCol + dCol;
    // Skip black cells.
    while (r >= 0 && r < grid.rows && c >= 0 && c < grid.cols) {
        if (!grid.cells[r][c].black) {
            selectCell(r, c);
            return;
        }
        r += dRow;
        c += dCol;
    }
}

function moveNext() {
    if (direction === "right") {
        moveSelection(0, 1);
    } else {
        moveSelection(1, 0);
    }
}

function movePrev() {
    if (direction === "right") {
        moveSelection(0, -1);
    } else {
        moveSelection(-1, 0);
    }
}

// --- Send move ---

async function sendMove(row, col, value) {
    // Optimistic update.
    state[row][col] = value;
    const td = getCell(row, col);
    if (td) td.textContent = value;

    try {
        const resp = await fetch(
            "/api/games/" + encodeURIComponent(gameID) + "/move",
            {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ pseudo, row, col, value }),
            }
        );
        if (!resp.ok) {
            // Revert on error.
            state[row][col] = "";
            if (td) td.textContent = "";
        }
    } catch {
        state[row][col] = "";
        if (td) td.textContent = "";
    }
}

// --- SSE ---

let reconnectDelay = 1000;
const maxReconnectDelay = 30000;

function connectSSE() {
    const statusEl = $("#connection-status");

    const url = "/api/games/" + encodeURIComponent(gameID) + "/events"
        + "?pseudo=" + encodeURIComponent(pseudo);

    eventSource = new EventSource(url);

    eventSource.onopen = () => {
        statusEl.hidden = true;
        reconnectDelay = 1000; // Reset on successful connect.
    };

    eventSource.onmessage = (e) => {
        const data = JSON.parse(e.data);

        if (data.type === "cell_update") {
            state[data.row][data.col] = data.value;
            const td = getCell(data.row, data.col);
            if (td) {
                td.textContent = data.value;
                // Flash animation for remote updates.
                if (data.pseudo !== pseudo) {
                    td.classList.add("cell-flash");
                    setTimeout(() => td.classList.remove("cell-flash"), 600);
                }
            }
        } else if (data.type === "player_joined") {
            addPlayerToList(data.pseudo, data.color);
        } else if (data.type === "player_left") {
            removePlayerFromList(data.pseudo);
        } else if (data.type === "game_state") {
            state = data.state;
            renderPlayers(data.players);
            refreshGridState();
        }
    };

    eventSource.onerror = () => {
        statusEl.hidden = false;
        eventSource.close();
        // Reconnect with exponential backoff.
        setTimeout(() => {
            reconnectDelay = Math.min(reconnectDelay * 2, maxReconnectDelay);
            connectSSE();
        }, reconnectDelay);
    };
}

function refreshGridState() {
    for (let r = 0; r < grid.rows; r++) {
        for (let c = 0; c < grid.cols; c++) {
            if (!grid.cells[r][c].black) {
                const td = getCell(r, c);
                if (td) td.textContent = state[r][c] || "";
            }
        }
    }
}

// --- Players ---

function renderPlayers(players) {
    const container = $("#player-list");
    container.textContent = "";
    if (!players) return;
    for (const key of Object.keys(players)) {
        addPlayerToList(players[key].pseudo, players[key].color);
    }
}

function removePlayerFromList(name) {
    const container = $("#player-list");
    const badge = container.querySelector('[data-pseudo="' + CSS.escape(name) + '"]');
    if (badge) {
        badge.classList.add("player-leaving");
        setTimeout(() => badge.remove(), 300);
    }
}

function addPlayerToList(name, color) {
    const container = $("#player-list");

    // Check if already listed.
    const existing = container.querySelector('[data-pseudo="' + CSS.escape(name) + '"]');
    if (existing) return;

    const badge = document.createElement("span");
    badge.className = "player-badge";
    badge.dataset.pseudo = name;
    badge.style.setProperty("--player-color", color);
    badge.textContent = name;
    container.appendChild(badge);
}

// --- Helpers ---

function showJoinError(msg) {
    clearJoinError();
    const p = document.createElement("p");
    p.className = "error-msg";
    p.textContent = msg;
    joinSection.appendChild(p);
}

function clearJoinError() {
    const el = joinSection.querySelector(".error-msg");
    if (el) el.remove();
}
