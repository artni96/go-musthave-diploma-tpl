package config

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
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
	TokenExp             time.Duration `env:"TOKEN_EXP"`
	Debug                bool          `env:"DEBUG"`
	UploadMechanics      bool          `env:"UPLOAD_MECHANICS"`
}

func ParseFlags() (*Config, error) {

	fs := flag.NewFlagSet("fs", flag.ExitOnError)
	config := &Config{}

	fs.StringVar(&config.RunAddress, "a", "localhost:8081", "run address")
	fs.StringVar(&config.DatabaseURI, "d", "", "database URI")
	fs.StringVar(&config.AccrualSystemAddress, "r", "http://localhost:8080", "accrual system address")
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

	envAccrualSystemAddress, ok := os.LookupEnv("ACCRUAL_SYSTEM_ADDRESS")
	if ok {
		config.AccrualSystemAddress = envAccrualSystemAddress
	}
	config.SecretKey = "secret"
	config.Debug = false
	config.TokenExp = time.Minute * 5
	config.UploadMechanics = false

	err = godotenv.Load(".env")
	if err == nil {

		envFileDBHost, envFileDBHostOk := os.LookupEnv("DB_HOST")
		envFileDBPort, envFileDBPortOk := os.LookupEnv("INNER_DB_PORT")
		envFileDBUser, envFileDBUserOk := os.LookupEnv("DB_USER")
		envFileDBPass, envFileDBPassOk := os.LookupEnv("DB_PASSWORD")
		enfFileDBName, enfFileDBNameOk := os.LookupEnv("DB_NAME")

		if envFileDBHostOk && envFileDBPortOk && envFileDBUserOk && envFileDBPassOk && enfFileDBNameOk {
			envFileDBDSN := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", envFileDBHost, envFileDBPort, envFileDBUser, envFileDBPass, enfFileDBName)
			config.DatabaseURI = envFileDBDSN
		}
		envFileAccrualSystemAddress, ok := os.LookupEnv("ACCRUAL_SYSTEM_ADDRESS")
		if ok {
			config.AccrualSystemAddress = envFileAccrualSystemAddress
		}

		envFileSecretKey, ok := os.LookupEnv("SECRET_KEY")
		if ok {
			config.SecretKey = envFileSecretKey
		}
		envFileTokenExp, ok := os.LookupEnv("TOKEN_EXP")
		if ok {
			config.TokenExp, err = time.ParseDuration(envFileTokenExp)
		}
		envFileUploadMechanics, ok := os.LookupEnv("UPLOAD_MECHANICS")
		if ok && strings.ToLower(envFileUploadMechanics) == "true" {
			config.UploadMechanics = true
		}
		envFileDebug, ok := os.LookupEnv("DEBUG")
		if ok && strings.ToLower(envFileDebug) == "true" {
			config.Debug = true
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
