package main

import (
	"delayedNotifier/url-shortener/internal/config"
	"delayedNotifier/url-shortener/internal/storage/postgres"
	"flag"
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

var (
	analyticsByDay          = flag.Bool("day", false, "Аналитика по дням")
	analyticsByMonth        = flag.Bool("month", false, "Аналитика по месяцам")
	analyticsByUserAgent    = flag.Bool("useragent", false, "Аналитика по User-Agent")
	analyticsRecentAccesses = flag.Bool("recent", false, "Недавние + лимит по ним")
	limit                   = flag.Int("limit", 0, "Лимит для недавних")
)

func main() {

	flag.Parse()
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
		IncludeDayStats:     *analyticsByDay,
		IncludeMonthStats:   *analyticsByMonth,
		IncludeUserAgent:    *analyticsByUserAgent,
		IncludeRecentAccess: *analyticsRecentAccesses,
		RecentAccessLimit:   *limit,
	}
	fmt.Println(optsAnalytics)
	analytics, err := storage.GetAnalyticsWithOptions("mylove", optsAnalytics)
	if err != nil {
		zlog.Logger.Error().Err(err).Msg("Ошибка получения аналитики")
	} else {
		zlog.Logger.Info().Msg("Аналитика получена")
		fmt.Printf("  Alias: %s\n", analytics.Alias)
		fmt.Printf("  URL: %s\n", analytics.URL)
		fmt.Printf("  Создан: %s\n", analytics.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("  Кликов: %d\n", analytics.Clicks)

		// flags
		if *analyticsByDay {
			for _, stat := range analytics.ByDay {
				fmt.Printf("  %s: %d кликов\n", stat.Date, stat.Clicks)
			}
		}
		if *analyticsByMonth {
			for _, stat := range analytics.ByMonth {
				fmt.Printf("  %s: %d кликов\n", stat.Month, stat.Clicks)
			}
		}
		if *analyticsByUserAgent {
			for _, stat := range analytics.ByUserAgent {
				fmt.Printf("  %s: %d кликов\n", stat.UserAgent, stat.Clicks)
			}
		}
		if *analyticsRecentAccesses {
			if *limit == 0 || *limit < 0 {
				*limit = 10
			}
			for _, stat := range analytics.RecentAccesses {
				fmt.Printf("User-Agent: %s, IP: %d, AccessedAt: %s, Referrer: %s", stat.UserAgent, stat.IPAddress, stat.AccessedAt, stat.Referrer)
			}
		}
	}

	//handlers
	//flags
	// will do redirect
}
