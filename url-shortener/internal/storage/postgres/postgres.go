package postgres

import (
	"context"
	"database/sql"
	"delayedNotifier/url-shortener/internal/storage"
	"fmt"
	"strings"
	"time"

	"github.com/wb-go/wbf/dbpg"
	"github.com/wb-go/wbf/retry"
	"github.com/wb-go/wbf/zlog"
)

type Storage struct {
	*dbpg.DB
}

type AnalyticsOptions struct {
	IncludeDayStats     bool
	IncludeMonthStats   bool
	IncludeUserAgent    bool
	IncludeRecentAccess bool
	RecentAccessLimit   int
	DaysLimit           int
	MonthsLimit         int
	UserAgentLimit      int
}

func NewStorage(masterDSN string, slaveDSNs []string) (*Storage, error) {
	const op = "postgres.NewStorage"
	opts := &dbpg.Options{
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
	}
	db, err := dbpg.New(masterDSN, slaveDSNs, opts)
	if err != nil {
		return nil, fmt.Errorf("%s : failed to connect to db: %w", op, err)
	}
	return &Storage{DB: db}, nil
}

func (s *Storage) CreateTable() error {
	const op = "postgres.Storage.CreateTable"

	ctx := context.Background()
	strategy := retry.Strategy{
		Attempts: 3,
		Delay:    100 * time.Millisecond,
		Backoff:  2,
	}
	// две таблицы для urls и аналитики short urls
	query := `
		CREATE TABLE IF NOT EXISTS urls(
		    id SERIAL PRIMARY KEY,
		    alias TEXT NOT NULL UNIQUE,
		    url TEXT NOT NULL,
		    created_at TIMESTAMP DEFAULT NOW(),
		    clicks INTEGER DEFAULT 0
		    );
		CREATE INDEX IF NOT EXISTS idx_alias ON urls(alias);

		CREATE TABLE IF NOT EXISTS url_analytics(
		    id SERIAL PRIMARY KEY,
            alias TEXT NOT NULL,
            accessed_at TIMESTAMP DEFAULT NOW(),
            user_agent TEXT,
            ip_address INET,
            referrer TEXT,
            CONSTRAINT fk_alias 
                FOREIGN KEY(alias) 
                REFERENCES urls(alias) 
                ON DELETE CASCADE
		);
 		CREATE INDEX IF NOT EXISTS idx_analytics_alias ON url_analytics(alias);
        CREATE INDEX IF NOT EXISTS idx_analytics_accessed_at ON url_analytics(accessed_at);
        CREATE INDEX IF NOT EXISTS idx_analytics_user_agent ON url_analytics(user_agent);
`
	_, err := s.ExecWithRetry(ctx, strategy, query)
	if err != nil {
		return fmt.Errorf("%s: failed to create table %w", op, err)
	}
	return nil
}

func (s *Storage) SaveURL(urlToSave, alias string) error {
	const op = "postgres.Storage.SaveURL"

	ctx := context.Background()
	strategy := retry.Strategy{
		Attempts: 3,
		Delay:    100 * time.Millisecond,
		Backoff:  2,
	}
	query := `
		INSERT INTO urls(alias, url) VALUES ($1, $2)
		`
	_, err := s.ExecWithRetry(ctx, strategy, query, alias, urlToSave)
	if err != nil {
		errStr := err.Error()

		isUniqueError := strings.Contains(errStr, "duplicate key") ||
			strings.Contains(errStr, "already exists") ||
			strings.Contains(errStr, "23505")

		if isUniqueError {
			return fmt.Errorf("%s: alias '%s' already exists: %w", op, alias, storage.ErrURLExists)
		}
		return fmt.Errorf("%s: failed to save url: %w", op, err)
	}
	return nil
}

func (s *Storage) GetURL(alias, userAgent, ipAddress, referrer string) (string, error) { //подумать как именно прикрутить счетчик(просто доставать здесь и инкремент)
	const op = "postgres.Storage.GetURL"

	ctx := context.Background()

	strategy := retry.Strategy{
		Attempts: 3,
		Delay:    100 * time.Millisecond,
		Backoff:  2,
	}
	var url string
	query := `
		SELECT url FROM urls WHERE alias = $1
		`
	row, err := s.QueryRowWithRetry(ctx, strategy, query, alias)
	if err != nil {
		return "", fmt.Errorf("%s failed to get url: %w", op, err)
	}
	err = row.Scan(&url)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("%s: alias `%s` %w", op, alias, storage.ErrURLNotFound)
		}
		return "", fmt.Errorf("%s: failed to scan url: %w", op, err)
	}
	//делаем синхронно увеличение счетчика
	err = s.incrementClicks(alias)
	if err != nil {
		// логируем здесь, чтобы не прерывать программу
		zlog.Logger.Warn().Err(err).Str("alias", alias).Msg("Не удалось увеличить счетчик кликов")
		//return "", fmt.Errorf("%s", err) // проверить как она выводится или так, чтобы прерывать
	}

	err = s.saveAnalytics(alias, userAgent, ipAddress, referrer)
	if err != nil {
		//тут бы я лучше прервал
		return "", fmt.Errorf("%s", err)
	}
	return url, nil
}

func (s *Storage) incrementClicks(alias string) error {
	const op = "postgres.Storage.incrementClicks"

	ctx := context.Background()
	query := `UPDATE urls SET clicks = clicks + 1 WHERE alias = $1`

	_, err := s.ExecContext(ctx, query, alias)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (s *Storage) saveAnalytics(alias, userAgent, ipAddress, referrer string) error {
	const op = "postgres.Storage.saveAnalytics"

	ctx := context.Background()
	query := `
		INSERT INTO url_analytics(alias,user_agent,ip_address, referrer)
		VALUES ($1,$2,$3,$4)
`
	_, err := s.ExecContext(ctx, query, alias, userAgent, ipAddress, referrer)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (s *Storage) GetAnalyticsWithOptions(alias string, opts AnalyticsOptions) (*storage.Analytics, error) {
	// число переходов просто запрос в бд
	// user-agent я бы брал с самого запроса в хендлере(другого не знать)
	// время переходов ну надо будто доп поле в бд (сука)
	const op = "postgres.Storage.GetAnalytics"
	ctx := context.Background()
	strategy := retry.Strategy{
		Attempts: 3,
		Delay:    100 * time.Millisecond,
		Backoff:  2,
	}

	query := `
		SELECT alias,url,created_at,clicks
		FROM urls
		WHERE alias = $1
`
	var analytics storage.Analytics
	row, err := s.QueryRowWithRetry(ctx, strategy, query, alias)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to query analytics: %w", op, err)
	}
	err = row.Scan(&analytics.Alias, &analytics.URL, &analytics.CreatedAt, &analytics.Clicks)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("%s: alias `%s` %w", op, alias, storage.ErrURLNotFound)
		}
		return nil, fmt.Errorf("%s: failed to scan url: %w", op, err)
	}

	// Инициализируем пустые слайсы (чтобы не было nil)
	analytics.ByDay = []storage.DayStats{}
	analytics.ByMonth = []storage.MonthStats{}
	analytics.ByUserAgent = []storage.UserAgentStats{}
	analytics.RecentAccesses = []storage.AccessDetail{}

	// Получаем детальную аналитику
	if opts.IncludeDayStats {
		limit := opts.DaysLimit
		if limit <= 0 {
			limit = 30 // default
		}
		dayStats, err := s.getAnalyticsByDay(alias)
		if err != nil {
			zlog.Logger.Warn().Err(err).Str("alias", alias).Msg("Ошибка получения аналитики по дням")
		} else {
			analytics.ByDay = dayStats
		}
	}
	if opts.IncludeMonthStats {
		limit := opts.MonthsLimit
		if limit <= 0 {
			limit = 12 // default
		}
		monthStats, err := s.getAnalyticsByMonth(alias)
		if err != nil {
			zlog.Logger.Warn().Err(err).Str("alias", alias).Msg("Ошибка получения аналитики по месяцам")
		} else {
			analytics.ByMonth = monthStats
		}
	}
	if opts.IncludeUserAgent {
		limit := opts.UserAgentLimit
		if limit <= 0 {
			limit = 20 // default
		}
		userAgentStats, err := s.getAnalyticsByUserAgent(alias)
		if err != nil {
			zlog.Logger.Warn().Err(err).Str("alias", alias).Msg("Ошибка получения аналитики по User-Agent")
		} else {
			analytics.ByUserAgent = userAgentStats
		}
	}
	if opts.IncludeRecentAccess {
		limit := opts.RecentAccessLimit
		if limit <= 0 {
			limit = 10 //default
		}
		recentAccesses, err := s.getRecentAccesses(alias, limit)
		if err != nil {
			zlog.Logger.Warn().Err(err).Str("alias", alias).Msg("Ошибка получения недавних переходов")
		} else {
			analytics.RecentAccesses = recentAccesses
		}
	}

	return &analytics, nil
}

func (s *Storage) getAnalyticsByDay(alias string) ([]storage.DayStats, error) {
	const op = "postgres.Storage.getAnalyticsByDay"

	ctx := context.Background()
	query := `
		SELECT
		DATE(accessed_at) as day,
		COUNT(*) as clicks
		FROM url_analytics
		WHERE alias = $1
		GROUP BY DATE(accessed_at)
        ORDER BY day DESC
		LIMIT 30
`
	rows, err := s.QueryContext(ctx, query, alias)
	if err != nil {
		return nil, fmt.Errorf("%s : %w", op, err)
	}
	defer rows.Close()

	var stats []storage.DayStats
	for rows.Next() {
		var stat storage.DayStats
		err := rows.Scan(&stat.Date, &stat.Clicks)
		if err != nil {
			return nil, fmt.Errorf("%s : %w", op, err)
		}
		stats = append(stats, stat)
	}
	return stats, rows.Err()
}

func (s *Storage) getAnalyticsByMonth(alias string) ([]storage.MonthStats, error) {
	ctx := context.Background()

	query := `
        SELECT 
            TO_CHAR(accessed_at, 'YYYY-MM') as month,
            COUNT(*) as clicks
        FROM url_analytics
        WHERE alias = $1
        GROUP BY TO_CHAR(accessed_at, 'YYYY-MM')
        ORDER BY month DESC
        LIMIT 12
    `

	rows, err := s.QueryContext(ctx, query, alias)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []storage.MonthStats
	for rows.Next() {
		var stat storage.MonthStats
		err := rows.Scan(&stat.Month, &stat.Clicks)
		if err != nil {
			return nil, err
		}
		stats = append(stats, stat)
	}

	return stats, rows.Err()
}

func (s *Storage) getAnalyticsByUserAgent(alias string) ([]storage.UserAgentStats, error) {
	ctx := context.Background()

	query := `
        SELECT 
            COALESCE(user_agent, 'Unknown') as user_agent,
            COUNT(*) as clicks
        FROM url_analytics
        WHERE alias = $1
        GROUP BY user_agent
        ORDER BY clicks DESC
        LIMIT 20
    `

	rows, err := s.QueryContext(ctx, query, alias)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []storage.UserAgentStats
	for rows.Next() {
		var stat storage.UserAgentStats
		err := rows.Scan(&stat.UserAgent, &stat.Clicks)
		if err != nil {
			return nil, err
		}
		stats = append(stats, stat)
	}

	return stats, rows.Err()
}

func (s *Storage) getRecentAccesses(alias string, limit int) ([]storage.AccessDetail, error) {
	ctx := context.Background()

	query := `
        SELECT 
            accessed_at,
            user_agent,
            ip_address,
            referrer
        FROM url_analytics
        WHERE alias = $1
        ORDER BY accessed_at DESC
        LIMIT $2
    `

	rows, err := s.QueryContext(ctx, query, alias, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accesses []storage.AccessDetail
	for rows.Next() {
		var access storage.AccessDetail
		err := rows.Scan(&access.AccessedAt, &access.UserAgent, &access.IPAddress, &access.Referrer)
		if err != nil {
			return nil, err
		}
		accesses = append(accesses, access)
	}

	return accesses, rows.Err()
}
