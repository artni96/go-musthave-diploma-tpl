package config

import (
	"fmt"
	"os"
)

func TestsDBDSN() string {
	host := os.Getenv("TEST_DB_HOST")
	port := os.Getenv("TEST_DB_PORT")
	user := os.Getenv("TEST_DB_USER")
	password := os.Getenv("TEST_DB_PASSWORD")
	dbname := os.Getenv("TEST_DB_NAME")

	var addr string
	if host == "" || port == "" || user == "" || password == "" || dbname == "" {
		addr = "host=localhost port=6432 user=test password=test dbname=gophermart_test sslmode=disable"
	} else {
		addr = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
	}

	return addr
}
