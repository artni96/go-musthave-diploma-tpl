package main

import (
	"log"

	"github.com/artni96/go-musthave-diploma-tpl/internal/gophermart/config"

	_ "github.com/artni96/go-musthave-diploma-tpl/api/docs"
)

// @title           Gophermart
// @version         1.0
// @description     This is HTTP API that provides main features of Gophermart loyalty program:\n    -user registration, authentification and authorization;\n    -order registration; \n   -user order list handling; \n    -user loyalty system management;\n    -registered user orders management
// @host            localhost:8080
// @BasePath        /
func main() {
	cfg, err := config.ParseFlags()
	if err != nil {
		log.Fatal(err)
	}
	if err = run(cfg); err != nil {
		log.Fatal(err)
	}
}
