package main

import (
	"fmt"
	"os"
	"syscall"

	"github.com/LamThanhNguyen/future-bank/util"
	"github.com/rs/zerolog/log"
)

var interruptSignals = []os.Signal{
	os.Interrupt,
	syscall.SIGTERM,
	syscall.SIGINT,
}

func main() {
	config, err := util.LoadConfig(".")
	if err != nil {
		log.Fatal().Err(err).Msg("cannot load config")
	}

	fmt.Printf("Loaded config: %+v\n", config)
	log.Info().Interface("config", config).Msg("loaded config")
}
