package storage

import "time"

type Analytics struct {
	Alias     string    `json:"alias"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"created_at"`
	Clicks    int       `json:"clicks"` // общее количество кликов

	// Детальная аналитика
	ByDay          []DayStats       `json:"by_day,omitempty"`
	ByMonth        []MonthStats     `json:"by_month,omitempty"`
	ByUserAgent    []UserAgentStats `json:"by_user_agent,omitempty"`
	RecentAccesses []AccessDetail   `json:"recent_accesses,omitempty"`
}
type DayStats struct {
	Date   string `json:"date"`
	Clicks int    `json:"clicks"`
}

type MonthStats struct {
	Month  string `json:"month"`
	Clicks int    `json:"clicks"`
}

type UserAgentStats struct {
	UserAgent string `json:"user_agent"`
	Clicks    int    `json:"clicks"`
}

type AccessDetail struct {
	AccessedAt time.Time `json:"accessed_at"`
	UserAgent  string    `json:"user_agent,omitempty"`
	IPAddress  string    `json:"ip_address,omitempty"`
	Referrer   string    `json:"referrer,omitempty"`
}
