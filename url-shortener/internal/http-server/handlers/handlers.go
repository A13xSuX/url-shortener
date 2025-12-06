package handlers

import (
	"delayedNotifier/url-shortener/internal/storage/postgres"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/wb-go/wbf/zlog"
)

type Handler struct {
	Storage *postgres.Storage
}

type ShortenRequest struct {
	URL   string `json:"url"`
	Alias string `json:"alias,omitempty"`
}

type ShortenResponse struct {
	Alias string `json:"alias"`
	URL   string `json:"url"`
}

func (h *Handler) ShortenURL(w http.ResponseWriter, r *http.Request) {
	// where are we need to logging
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req ShortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.URL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}
	// maybe add playgroundvalidator
	alias := req.Alias
	if alias == "" {
		alias = generateRandomAlias(5)
	}

	err := h.Storage.SaveURL(req.URL, alias)
	if err != nil {
		// точно ли already exists, в main точно правильно должно быть или в save поискать
		if strings.Contains(err.Error(), "already exists") {
			http.Error(w, "Alias already exists", http.StatusConflict)
		}
		http.Error(w, "Failed to save URL", http.StatusInternalServerError)
		return
	}

	resp := ShortenResponse{
		Alias: alias,
		URL:   req.URL,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	//как будто можно ошибку обработать
	if err = json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "Invalid response body", http.StatusInternalServerError)
		return
	}
	//почему где то str с ключом а где msg можно же msgf
	zlog.Logger.Info().Str("alias", alias).Str("url", req.URL).Msg("URL успешно сохранен")
}
