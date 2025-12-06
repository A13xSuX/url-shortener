package handlers

import (
	"delayedNotifier/url-shortener/internal/config"
	"delayedNotifier/url-shortener/internal/storage/postgres"
	"encoding/json"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/wb-go/wbf/zlog"
)

type Handler struct {
	Storage *postgres.Storage
	Config  *config.AppConfig
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

func (h *Handler) RedirectURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	//extract alias
	path := strings.TrimPrefix(r.URL.Path, "/s/") //remeber what to do TrimPrefix
	if path == "" {
		http.Error(w, "Short URL not provided", http.StatusBadRequest)
		return
	}
	alias := path
	// Получаем User-Agent, IP и Referrer
	userAgent := r.UserAgent()
	ipAddress := getClientIP(r)
	referrer := r.Referer()

	//get orig url
	url, err := h.Storage.GetURL(alias, userAgent, ipAddress, referrer)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Short URL not found", http.StatusNotFound)
			return
		}
		zlog.Logger.Error().Err(err).Msg("Short URL not found")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, url, http.StatusFound)

	zlog.Logger.Info().Str("alias", alias).Str("url", url).Str("ip", ipAddress).Msg("Редирект выполнен")
}

func generateRandomAlias(length int) string {
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	chars := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
		"abcdefghijklmnopqrstuvwxyz" +
		"0123456789")
	result := make([]rune, length)
	for i := range result {
		result[i] = chars[rnd.Intn(len(chars))]
	}
	return string(result)
}

func getClientIP(r *http.Request) string {
	ip := r.Header.Get("X-Forwarded-For")
	if ip != "" {
		return ip
	}
	ip = r.Header.Get("X-Real-IP")
	if ip != "" {
		return ip
	}
	// Если нет заголовков, используем RemoteAddr
	return strings.Split(r.RemoteAddr, ":")[0] //что делает
}
