package config

import (
	"fmt"
	"time"

	"github.com/wb-go/wbf/config"
)

type AppConfig struct {
	ServerConfig    serverConfig
	LoggerConfig    loggerConfig
	PostgresConfig  postgresConfig
	AnalyticsConfig AnalyticsConfig
}

type serverConfig struct {
	Address string
	//timeout     int
	//idleTimeout int
}

type loggerConfig struct {
	LogLevel string
}

type postgresConfig struct {
	MaxOpenConnections int
	MaxIdleConnections int
	ConnMaxLifetime    time.Duration
	Port               int
}

// AnalyticsIncludeConfig - что включать в аналитику
type AnalyticsIncludeConfig struct {
	Days           bool `yaml:"days"`
	Months         bool `yaml:"months"`
	UserAgent      bool `yaml:"user_agent"`
	RecentAccesses bool `yaml:"recent_accesses"`
}

// AnalyticsLimitsConfig - лимиты для аналитики
type AnalyticsLimitsConfig struct {
	RecentAccesses int `yaml:"recent_accesses"`
	Days           int `yaml:"days"`
	Months         int `yaml:"months"`
	UserAgents     int `yaml:"user_agents"`
}

type AnalyticsConfig struct {
	Include AnalyticsIncludeConfig `yaml:"include"`
	Limit   AnalyticsLimitsConfig  `yaml:"limit"`
}

func NewAppConfig() (*AppConfig, error) {
	envFilePath := "../../.env"
	appConfigFilePath := "../../config/local.yaml" //do one config instead two

	cfg := config.New()

	// Загрузка .env файлов
	if err := cfg.LoadEnvFiles(envFilePath); err != nil {
		return nil, fmt.Errorf("failed to load env files: %w", err)
	}

	// Включение поддержки переменных окружения
	cfg.EnableEnv("")

	// Загрузка файлов конфигурации
	if err := cfg.LoadConfigFiles(appConfigFilePath); err != nil {
		return nil, fmt.Errorf("failed to load config files: %w", err)
	}

	// Определение флагов командной строки
	cfg.DefineFlag("p", "srvport", "transport.http.port", 7777, "HTTP server port")
	if err := cfg.ParseFlags(); err != nil {
		return nil, fmt.Errorf("failed to pars flags: %w", err)
	}
	var appConfig AppConfig
	appConfig.ServerConfig.Address = cfg.GetString("server.address")
	appConfig.LoggerConfig.LogLevel = cfg.GetString("logger.level")
	//appConfig.loggerConfig.logLevel = cfg.GetString("logger.level")
	appConfig.PostgresConfig.MaxIdleConnections = cfg.GetInt("postgres.max_idle_connections")
	appConfig.PostgresConfig.MaxOpenConnections = cfg.GetInt("postgres.max_open_connections")
	appConfig.PostgresConfig.ConnMaxLifetime = cfg.GetDuration("postgres.conn_max_lifetime")
	appConfig.PostgresConfig.Port = cfg.GetInt("POSTGRES_PORT") // из переменной окружения (из файла .env)
	appConfig.AnalyticsConfig.Include.Days = cfg.GetBool("analytics.include.days")
	appConfig.AnalyticsConfig.Include.Months = cfg.GetBool("analytics.include.months")
	appConfig.AnalyticsConfig.Include.UserAgent = cfg.GetBool("analytics.include.user_agent")
	appConfig.AnalyticsConfig.Include.RecentAccesses = cfg.GetBool("analytics.include.recent_accesses")
	appConfig.AnalyticsConfig.Limit.RecentAccesses = cfg.GetInt("analytics.limits.recent_accesses")
	appConfig.AnalyticsConfig.Limit.Days = cfg.GetInt("analytics.limits.days")
	appConfig.AnalyticsConfig.Limit.Months = cfg.GetInt("analytics.limits.months")
	appConfig.AnalyticsConfig.Limit.UserAgents = cfg.GetInt("analytics.limits.user_agents")

	return &appConfig, nil
}
