package main

import (
	"os"

	"github.com/joeshaw/envdecode"
	"github.com/porter-dev/porter-agent/internal/adapter"
	"github.com/porter-dev/porter-agent/internal/envconf"
	"github.com/porter-dev/porter-agent/internal/logger"
	"github.com/porter-dev/porter-agent/internal/repository"
)

func main() {
	envDecoderConf := envconf.EnvDecoderConf{}

	if err := envdecode.StrictDecode(&envDecoderConf); err != nil {
		logger.NewErrorConsole(true).Fatal().Caller().Msgf("could not decode env conf: %v", err)
		os.Exit(1)
	}

	l := logger.NewConsole(envDecoderConf.Debug)
	db, err := adapter.New(&envDecoderConf.DBConf)

	if err != nil {
		l.Fatal().Caller().Msgf("could not create database connection: %v", err)
	}

	err = repository.AutoMigrate(db, false)

	if err != nil {
		l.Fatal().Caller().Msgf("auto migration failed: %v", err)
	}

	repo := repository.NewRepository(db)

	repo.Event.DeleteOlderEvents(l)
	repo.EventCache.DeleteOlderEventCaches(l)
}
