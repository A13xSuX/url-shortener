package handlers

import (
	"delayedNotifier/url-shortener/internal/config"
	"delayedNotifier/url-shortener/internal/storage/postgres"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/wb-go/wbf/zlog"
)

func (h *Handler) GetAnalytics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/analytics")
	if path == "" || path == "/analytics" {
		http.Error(w, "Short URL not provided", http.StatusBadRequest)
		return
	}
	//посмотреть в local config ну так мало ли
	alias := path
	query := r.URL.Query()
	includeDays := query.Get("days") == "true"
	includeMonths := query.Get("months") == "true"
	includeUA := query.Get("ua") == "true"
	includeRecent := query.Get("recent") == "true"

	limitRecent := 10
	if l := query.Get("limit_recent"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limitRecent = n
		}
	}

	limitDays := 30
	if l := query.Get("limit_days"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limitDays = n
		}
	}

	limitMonths := 12
	if l := query.Get("limit_months"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limitMonths = n
		}
	}

	limitUA := 20
	if l := query.Get("limit_ua"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limitUA = n
		}
	}

	// Используем конфигурацию или параметры запроса
	cfg, err := config.NewAppConfig()
	var optsAnalytics postgres.AnalyticsOptions

	if err == nil {
		// Используем конфигурацию, но переопределяем параметрами запроса
		optsAnalytics = postgres.AnalyticsOptions{
			IncludeDayStats:     cfg.AnalyticsConfig.Include.Days || includeDays,
			IncludeMonthStats:   cfg.AnalyticsConfig.Include.Months || includeMonths,
			IncludeUserAgent:    cfg.AnalyticsConfig.Include.UserAgent || includeUA,
			IncludeRecentAccess: cfg.AnalyticsConfig.Include.RecentAccesses || includeRecent,
			RecentAccessLimit:   cfg.AnalyticsConfig.Limit.RecentAccesses,
			DaysLimit:           cfg.AnalyticsConfig.Limit.Days,
			MonthsLimit:         cfg.AnalyticsConfig.Limit.Months,
			UserAgentLimit:      cfg.AnalyticsConfig.Limit.UserAgents,
		}
	} else {
		// Используем только параметры запроса
		optsAnalytics = postgres.AnalyticsOptions{
			IncludeDayStats:     includeDays,
			IncludeMonthStats:   includeMonths,
			IncludeUserAgent:    includeUA,
			IncludeRecentAccess: includeRecent,
			RecentAccessLimit:   limitRecent,
			DaysLimit:           limitDays,
			MonthsLimit:         limitMonths,
			UserAgentLimit:      limitUA,
		}
	}

	// Переопределяем лимиты если заданы в запросе
	if includeRecent && query.Get("limit_recent") != "" {
		optsAnalytics.RecentAccessLimit = limitRecent
	}
	if includeDays && query.Get("limit_days") != "" {
		optsAnalytics.DaysLimit = limitDays
	}
	if includeMonths && query.Get("limit_months") != "" {
		optsAnalytics.MonthsLimit = limitMonths
	}
	if includeUA && query.Get("limit_ua") != "" {
		optsAnalytics.UserAgentLimit = limitUA
	}
	//муть

	analytics, err := h.Storage.GetAnalyticsWithOptions(alias, optsAnalytics)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Short URL not found", http.StatusNotFound)
			return
		}
		zlog.Logger.Error().Err(err).Msg("Ошибка получения аналитики")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(analytics)
	if err != nil {
		http.Error(w, "failed encode", http.StatusInternalServerError)
		return
	}
	zlog.Logger.Info().Str("alias", alias).Msg("Аналитика отправлена")
}
