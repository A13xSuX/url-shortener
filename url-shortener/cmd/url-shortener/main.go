package main

import (
	"delayedNotifier/url-shortener/internal/config"
	"delayedNotifier/url-shortener/internal/storage/postgres"
	"fmt"
	"log"

	_ "github.com/lib/pq"
	"github.com/wb-go/wbf/dbpg"
	"github.com/wb-go/wbf/zlog"
)

//config +
//log +
//storage
//logic
//handlers
//servers

func main() {

	//cfg
	cfg, err := config.NewAppConfig()
	if err != nil {
		log.Fatalln("Pizda")
	}
	//logger
	zlog.InitConsole()
	//set level of debug ("trace", "debug", "info", "warn", "error", "fatal", "panic".)
	//поменять потом в конструцию switch,
	//в зависимости от параметра env dev prod local
	err = zlog.SetLevel(cfg.LoggerConfig.LogLevel)
	if err != nil {
		log.Fatalf("Level of Logger invalid: %s", cfg.LoggerConfig.LogLevel)
	}
	_ = cfg

	//мб вынести это тоже в postgres go
	opts := &dbpg.Options{MaxOpenConns: cfg.PostgresConfig.MaxOpenConnections, MaxIdleConns: cfg.PostgresConfig.MaxIdleConnections}
	//masterDSN — строка подключения к мастер-узлу(will check)
	// могу ли я как то поменять на localhost и чтобы тоже работало
	masterDSN := "host=host.docker.internal port=5432 user=postgres password=postgres dbname=mydb sslmode=disable"
	//slaveDSNs — массив строк подключения к slave-узлам у меня нет реплик
	db, err := dbpg.New(masterDSN, []string{}, opts)
	if err != nil {
		zlog.Logger.Debug().Err(err)
	}
	defer db.Master.Close()
	zlog.Logger.Info().Msgf("Success connection to db : %v", db)

	//будто надо new сделать и только storage оставлять в main
	storage := &postgres.Storage{DB: db}
	err = storage.CreateTable()
	if err != nil {
		zlog.Logger.Error().Err(err).Msgf("Ошибка создания таблицы %w", err)
	} else {
		fmt.Println("Таблица создана успешно")
	}

	//создание
	//err = storage.SaveURL("https://www.twitch.tv/", "mylove")
	//if err != nil {
	//	zlog.Logger.Error().Err(err).Msg("Ошибка сохранения URL")
	//}

	//get
	url, err := storage.GetURL("mylove", "test-agent", "127.0.0.1", "")
	if err != nil {
		zlog.Logger.Error().Err(err).Msg("Не нашел почтально печкин")
	} else {
		zlog.Logger.Info().Msgf("нашли прикинь: %s", url)
	}

	optsAnalytics := postgres.AnalyticsOptions{
		IncludeDayStats:     cfg.AnalyticsConfig.Include.Days,
		IncludeMonthStats:   cfg.AnalyticsConfig.Include.Months,
		IncludeUserAgent:    cfg.AnalyticsConfig.Include.UserAgent,
		IncludeRecentAccess: cfg.AnalyticsConfig.Include.RecentAccesses,
		RecentAccessLimit:   cfg.AnalyticsConfig.Limit.RecentAccesses,
		DaysLimit:           cfg.AnalyticsConfig.Limit.Days,
		MonthsLimit:         cfg.AnalyticsConfig.Limit.Months,
		UserAgentLimit:      cfg.AnalyticsConfig.Limit.UserAgents,
	}
	//отладка
	fmt.Printf("\nПараметры аналитики:\n")
	fmt.Printf("  Включить дни: %v (лимит: %d)\n",
		optsAnalytics.IncludeDayStats, optsAnalytics.DaysLimit)
	fmt.Printf("  Включить месяцы: %v (лимит: %d)\n",
		optsAnalytics.IncludeMonthStats, optsAnalytics.MonthsLimit)
	fmt.Printf("  Включить User-Agent: %v (лимит: %d)\n",
		optsAnalytics.IncludeUserAgent, optsAnalytics.UserAgentLimit)
	fmt.Printf("  Включить недавние переходы: %v (лимит: %d)\n",
		optsAnalytics.IncludeRecentAccess, optsAnalytics.RecentAccessLimit)
	//отладка
	analytics, err := storage.GetAnalyticsWithOptions("mylove", optsAnalytics)
	if err != nil {
		zlog.Logger.Error().Err(err).Msg("Ошибка получения аналитики")
	} else {
		zlog.Logger.Info().Msg("Аналитика получена")
		fmt.Printf("  Alias: %s\n", analytics.Alias)
		fmt.Printf("  URL: %s\n", analytics.URL)
		fmt.Printf("  Создан: %s\n", analytics.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("  Кликов: %d\n", analytics.Clicks)

		//доп аналитика(доделать мб и логировать бы в инфо
		if optsAnalytics.IncludeDayStats {
			fmt.Println("вывод по дням")
			for _, stat := range analytics.ByDay {
				fmt.Printf("  %s: %d кликов\n", stat.Date, stat.Clicks)
			}
		}
		if optsAnalytics.IncludeMonthStats {
			fmt.Println("вывод по месяцам")
			for _, stat := range analytics.ByMonth {
				fmt.Printf("  %s: %d кликов\n", stat.Month, stat.Clicks)
			}
		}
		if optsAnalytics.IncludeUserAgent {
			fmt.Println("вывод по юзерагенту")
			for _, stat := range analytics.ByUserAgent {
				fmt.Printf("  %s: %d кликов\n", stat.UserAgent, stat.Clicks)
			}
		}
		if optsAnalytics.IncludeRecentAccess {
			fmt.Println("вывод по недавни")
			for _, stat := range analytics.RecentAccesses {
				fmt.Printf("  %s\n %s\n  %s\n %s\n", stat.AccessedAt.Format("2006-01-02 15:04:05"),
					stat.UserAgent, stat.IPAddress, stat.Referrer)
			}
		}
	}

	//handlers
	//flags
	// will do redirect
}
