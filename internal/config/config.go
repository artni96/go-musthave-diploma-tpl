package config

import (
	"errors"
	"flag"
	"log"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

type Config struct {
	RunAddress           string        `env:"RUN_ADDRESS"`
	DatabaseURI          string        `env:"DATABASE_URI"`
	AccrualSystemAddress string        `env:"ACCRUAL_SYSTEM_ADDRESS"`
	SecretKey            string        `env:"SECRET_KEY"`
	TokenExp             time.Duration `env:"TOKEN_EXPIRATION"`
	LoggerLevel          string        `env:"LOGGER_LEVEL"`
}

func ParseFlags() (*Config, error) {

	fs := flag.NewFlagSet("fs", flag.ExitOnError)
	config := &Config{}

	fs.StringVar(&config.RunAddress, "a", "localhost:8081", "run address")
	fs.StringVar(&config.DatabaseURI, "d", "", "database URI")
	fs.StringVar(&config.AccrualSystemAddress, "r", "localhost:8080", "accrual system address")
	fs.StringVar(&config.LoggerLevel, "l", "Info", "log level")

	err := fs.Parse(os.Args[1:])
	if err != nil {
		return nil, err
	}

	envRunAddress, ok := os.LookupEnv("RUN_ADDRESS")
	if ok {
		config.RunAddress = envRunAddress
	}

	envDatabaseURI, ok := os.LookupEnv("DATABASE_URI")
	if ok {
		config.DatabaseURI = envDatabaseURI
	}

	envAccrualSystemAddress, ok := os.LookupEnv("ACCRUAL_ADDRESS")
	if ok {
		config.AccrualSystemAddress = envAccrualSystemAddress
	}

	err = godotenv.Load(".env")
	if err != nil {
		log.Println(".env file not found")
		return config, errors.New(".env file not found - keep working with default values")
	}
	envSecretKey, ok := os.LookupEnv("SECRET_KEY")
	if ok {
		config.SecretKey = envSecretKey
	} else {
		config.SecretKey = "secret"
	}
	envTokenExp, ok := os.LookupEnv("TOKEN_EXPIRATION")
	if ok {
		config.TokenExp, err = time.ParseDuration(envTokenExp)
	} else {
		config.TokenExp = time.Minute * 5
	}
	envLoggerLevel, ok := os.LookupEnv("LOGGER_LEVEL")
	if ok {
		config.LoggerLevel = envLoggerLevel
	} else {
		config.LoggerLevel = "info"
	}

	return config, nil
}

type App struct {
	DB     *sqlx.DB
	Config *Config
	Logger *zap.Logger
}
