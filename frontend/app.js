"use strict";

const $ = (sel) => document.querySelector(sel);

// --- Upload ---

const fileInput = $("#file-input");
const btnUpload = $("#btn-upload");
const uploadStatus = $("#upload-status");

btnUpload.addEventListener("click", () => fileInput.click());

fileInput.addEventListener("change", async () => {
    const file = fileInput.files[0];
    if (!file) return;

    btnUpload.disabled = true;
    uploadStatus.hidden = false;
    clearError();

    const form = new FormData();
    form.append("image", file);

    try {
        const resp = await fetch("/api/grids", { method: "POST", body: form });
        if (!resp.ok) {
            const data = await resp.json();
            throw new Error(data.error || "Erreur inconnue");
        }
        const grid = await resp.json();
        renderGridPreview(grid);
        loadGridList();
    } catch (err) {
        showError(err.message);
    } finally {
        btnUpload.disabled = false;
        uploadStatus.hidden = true;
        fileInput.value = "";
    }
});

// --- Grid list ---

async function loadGridList() {
    const container = $("#grid-list");
    try {
        const resp = await fetch("/api/grids");
        const grids = await resp.json();

        container.textContent = "";

        if (!grids || grids.length === 0) {
            const p = document.createElement("p");
            p.className = "empty-state";
            p.textContent = "Aucune grille pour le moment.";
            container.appendChild(p);
            return;
        }

        for (const g of grids) {
            container.appendChild(createGridCard(g));
        }
    } catch {
        container.textContent = "";
        const p = document.createElement("p");
        p.className = "error-msg";
        p.textContent = "Impossible de charger les grilles.";
        container.appendChild(p);
    }
}

function createGridCard(g) {
    const date = new Date(g.created_at).toLocaleDateString("fr-FR", {
        day: "numeric", month: "short", hour: "2-digit", minute: "2-digit",
    });
    const defCount = countDefs(g);

    const card = document.createElement("div");
    card.className = "grid-card";

    const info = document.createElement("div");
    info.className = "grid-card-info";

    const title = document.createElement("span");
    title.className = "grid-card-title";
    title.textContent = "Grille " + g.rows + "\u00d7" + g.cols;

    const meta = document.createElement("span");
    meta.className = "grid-card-meta";
    meta.textContent = defCount + " d\u00e9finitions \u00b7 " + date;

    info.appendChild(title);
    info.appendChild(meta);

    const actions = document.createElement("div");
    actions.className = "grid-card-actions";

    const btnView = document.createElement("button");
    btnView.className = "btn btn-secondary";
    btnView.textContent = "Voir";
    btnView.addEventListener("click", () => showGrid(g.id));

    const btnPlay = document.createElement("button");
    btnPlay.className = "btn btn-primary";
    btnPlay.textContent = "Jouer";
    btnPlay.addEventListener("click", () => createGame(g.id));

    actions.appendChild(btnView);
    actions.appendChild(btnPlay);

    card.appendChild(info);
    card.appendChild(actions);
    return card;
}

async function showGrid(id) {
    try {
        const resp = await fetch("/api/grids/" + encodeURIComponent(id));
        if (!resp.ok) throw new Error();
        const grid = await resp.json();
        renderGridPreview(grid);
    } catch {
        showError("Impossible de charger la grille.");
    }
}

// --- Create game ---

async function createGame(gridID) {
    try {
        const resp = await fetch("/api/games", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ grid_id: gridID }),
        });
        if (!resp.ok) {
            const data = await resp.json();
            throw new Error(data.error || "Erreur");
        }
        const game = await resp.json();
        location.href = "/game/" + encodeURIComponent(game.id);
    } catch (err) {
        showError(err.message);
    }
}

// --- Grid rendering ---

function renderGridPreview(grid) {
    const section = $("#grid-preview");
    const table = $("#grid-table");
    table.textContent = "";

    for (const row of grid.cells) {
        const tr = document.createElement("tr");
        for (const cell of row) {
            const td = document.createElement("td");
            if (cell.black) {
                td.className = "cell-def";
                renderDefsInto(td, cell.definitions);
            } else {
                td.className = "cell-letter";
            }
            tr.appendChild(td);
        }
        table.appendChild(tr);
    }

    section.hidden = false;
    section.scrollIntoView({ behavior: "smooth", block: "start" });
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

// --- Helpers ---

function countDefs(grid) {
    let n = 0;
    for (const row of grid.cells) {
        for (const cell of row) {
            if (cell.black && cell.definitions) n += cell.definitions.length;
        }
    }
    return n;
}

function showError(msg) {
    clearError();
    const p = document.createElement("p");
    p.className = "error-msg";
    p.textContent = msg;
    $(".section-upload").appendChild(p);
}

function clearError() {
    const el = $(".section-upload .error-msg");
    if (el) el.remove();
}

// --- Init ---

loadGridList();
