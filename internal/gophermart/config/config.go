package config

import (
	"flag"
	"log"
	"os"
	"time"

	"github.com/artni96/go-musthave-diploma-tpl/api/docs"
	_ "github.com/artni96/go-musthave-diploma-tpl/api/docs"
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
	Debug                bool          `env:"DEBUG"`
	UploadMechanics      bool          `env:"UPLOAD_MECHANICS"`
}

func ParseFlags() (*Config, error) {

	fs := flag.NewFlagSet("fs", flag.ExitOnError)
	config := &Config{}

	fs.StringVar(&config.RunAddress, "a", "localhost:8081", "run address")
	fs.StringVar(&config.DatabaseURI, "d", "", "database URI")
	fs.StringVar(&config.AccrualSystemAddress, "r", "localhost:8080", "accrual system address")
	docs.SwaggerInfo.Host = config.RunAddress

	err := fs.Parse(os.Args[1:])
	if err != nil {
		return nil, err
	}

	envRunAddress, ok := os.LookupEnv("RUN_ADDRESS")
	if ok {
		config.RunAddress = envRunAddress
		docs.SwaggerInfo.Host = envRunAddress
	}

	envDatabaseURI, ok := os.LookupEnv("DATABASE_URI")
	if ok {
		config.DatabaseURI = envDatabaseURI
	}

	envAccrualSystemAddress, ok := os.LookupEnv("ACCRUAL_ADDRESS")
	if ok {
		config.AccrualSystemAddress = envAccrualSystemAddress
	}
	config.SecretKey = "secret"
	config.Debug = true
	config.TokenExp = time.Minute * 5
	config.UploadMechanics = false

	err = godotenv.Load(".env")
	if err == nil {
		envSecretKey, ok := os.LookupEnv("SECRET_KEY")
		if ok {
			config.SecretKey = envSecretKey
		}
		envTokenExp, ok := os.LookupEnv("TOKEN_EXPIRATION")
		if ok {
			config.TokenExp, err = time.ParseDuration(envTokenExp)
		}
		envDebug, ok := os.LookupEnv("DEBUG")
		if ok && envDebug == "true" {
			config.Debug = true
		}
		envUploadMechanics, ok := os.LookupEnv("UPLOAD_MECHANICS")
		if ok && envUploadMechanics == "true" {
			config.UploadMechanics = true
		}
	} else {
		log.Println(".env file not found, keep working with default values")
	}
	return config, nil
}

type App struct {
	DB     *sqlx.DB
	Config *Config
	Logger *zap.Logger
}
