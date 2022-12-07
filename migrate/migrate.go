package main

import (
	"github.com/porter-dev/porter-agent/internal/adapter"
	"github.com/porter-dev/porter-agent/internal/envconf"
	"github.com/porter-dev/porter-agent/internal/logger"
	"github.com/porter-dev/porter-agent/internal/repository"
)

func main() {
	envDecoderConf := envconf.EnvDecoderConf{}
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
