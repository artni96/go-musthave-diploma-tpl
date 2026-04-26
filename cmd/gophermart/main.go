package main

import (
	"log"

	"github.com/artni96/go-musthave-diploma-tpl/internal/config"
)

func main() {
	cfg, err := config.ParseFlags()
	if err != nil {
		log.Fatal(err)
	}
	if err = run(cfg); err != nil {
		log.Fatal(err)
	}
}
