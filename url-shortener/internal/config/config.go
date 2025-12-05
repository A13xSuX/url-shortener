package config

import (
	"fmt"
	"time"

	"github.com/wb-go/wbf/config"
)

type AppConfig struct {
	ServerConfig   serverConfig
	LoggerConfig   loggerConfig
	PostgresConfig postgresConfig
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

	return &appConfig, nil
}
