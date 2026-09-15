package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/viper"
)

type Config struct {
	Environment   string              `mapstructure:"environment"`
	Server        ServerConfig        `mapstructure:"server"`
	Database      DatabaseConfig      `mapstructure:"database"`
	ClassroomSync ClassroomSyncConfig `mapstructure:"classroom_sync"`
}

type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Name     string `mapstructure:"name"`
	SSLMode  string `mapstructure:"sslmode"`
}

type ClassroomSyncConfig struct {
	UserID int64 `mapstructure:"user_id"`
}

func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.Name, d.SSLMode,
	)
}

func (c Config) IsProduction() bool {
	return c.Environment == "production"
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	// 展开配置值中的 ${VAR}
	cfg.Database.Password = os.ExpandEnv(cfg.Database.Password)
	cfg.Database.User = os.ExpandEnv(cfg.Database.User)
	cfg.Database.Name = os.ExpandEnv(cfg.Database.Name)
	applyEnvironmentOverrides(&cfg)

	if cfg.Server.Port <= 0 {
		return nil, fmt.Errorf("server.port must be greater than zero")
	}
	if cfg.Server.Host == "" {
		return nil, fmt.Errorf("server.host is required")
	}
	if cfg.Database.Password == "" {
		return nil, fmt.Errorf("database.password is required")
	}
	if cfg.Environment != "development" && cfg.Environment != "production" {
		return nil, fmt.Errorf("environment must be development or production")
	}
	if cfg.ClassroomSync.UserID < 0 {
		return nil, fmt.Errorf("classroom_sync.user_id must not be negative")
	}

	return &cfg, nil
}

func applyEnvironmentOverrides(cfg *Config) {
	if value := os.Getenv("CAIRN_ENVIRONMENT"); value != "" {
		cfg.Environment = value
	}
	if value := os.Getenv("CAIRN_SERVER_HOST"); value != "" {
		cfg.Server.Host = value
	}
	if value := os.Getenv("CAIRN_SERVER_PORT"); value != "" {
		if port, err := strconv.Atoi(value); err == nil {
			cfg.Server.Port = port
		}
	}
	if value := os.Getenv("CAIRN_DATABASE_HOST"); value != "" {
		cfg.Database.Host = value
	}
	if value := os.Getenv("CAIRN_DATABASE_PORT"); value != "" {
		if port, err := strconv.Atoi(value); err == nil {
			cfg.Database.Port = port
		}
	}
	if value := os.Getenv("CAIRN_DATABASE_USER"); value != "" {
		cfg.Database.User = value
	}
	if value := os.Getenv("CAIRN_DATABASE_NAME"); value != "" {
		cfg.Database.Name = value
	}
	if value := os.Getenv("CAIRN_DATABASE_SSLMODE"); value != "" {
		cfg.Database.SSLMode = value
	}
	if value := os.Getenv("CAIRN_CLASSROOM_SYNC_USER_ID"); value != "" {
		if userID, err := strconv.ParseInt(value, 10, 64); err == nil {
			cfg.ClassroomSync.UserID = userID
		}
	}
}
