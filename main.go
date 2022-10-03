package main

import (
	"os"
	"time"

	"gorm.io/gorm"
	"k8s.io/client-go/kubernetes"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/porter-dev/porter-agent/internal/models"
	"github.com/porter-dev/porter/api/server/shared/config/env"

	"github.com/gin-gonic/gin"
	"github.com/joeshaw/envdecode"
	"github.com/porter-dev/porter-agent/internal/adapter"
	"github.com/porter-dev/porter-agent/internal/logger"
	"github.com/porter-dev/porter-agent/internal/repository"
	"github.com/porter-dev/porter-agent/pkg/alerter"
	"github.com/porter-dev/porter-agent/pkg/controllers"
	"github.com/porter-dev/porter-agent/pkg/httpclient"
	"github.com/porter-dev/porter-agent/pkg/incident"
	"github.com/porter-dev/porter-agent/pkg/logstore"
	"github.com/porter-dev/porter-agent/pkg/logstore/memorystore"
	"github.com/porter-dev/porter-agent/pkg/pulsar"
)

var (
	httpServer *gin.Engine
)

type LogStoreConf struct {
	LogStoreKind string `env:"LOG_STORE_KIND,default=memory"`

	// TODO: loki environment variables for initialization here
}
type EnvDecoderConf struct {
	Debug bool `env:"DEBUG,default=true"`

	LogStoreConf   LogStoreConf
	HTTPClientConf httpclient.HTTPClientConf
	DBConf         env.DBConf
}

func main() {
	kubeClient := kubernetes.NewForConfigOrDie(ctrl.GetConfigOrDie())

	var envDecoderConf EnvDecoderConf = EnvDecoderConf{}

	if err := envdecode.StrictDecode(&envDecoderConf); err != nil {
		logger.NewErrorConsole(true).Fatal().Caller().Msgf("could not decode env conf: %v", err)

		os.Exit(1)
	}

	l := logger.NewConsole(envDecoderConf.Debug)

	client := httpclient.NewClient(&envDecoderConf.HTTPClientConf, l)

	// create database connection through adapter
	db, err := adapter.New(&envDecoderConf.DBConf)

	if err != nil {
		l.Fatal().Caller().Msgf("could not create database connection: %v", err)
	}

	if err := repository.AutoMigrate(db, false); err != nil {
		l.Fatal().Caller().Msgf("auto migration failed: %v", err)
	}

	var logStore logstore.LogStore

	if envDecoderConf.LogStoreConf.LogStoreKind == "memory" {
		logStore, err = memorystore.New("test", memorystore.Options{})

		if err != nil {
			l.Fatal().Caller().Msgf("memory-based log store setup failed: %v", err)
		}
	} else {
		l.Fatal().Caller().Msg("loki integration not enabled")
	}

	go cleanupEventCache(db, l)

	repo := repository.NewRepository(db)

	alerter := &alerter.Alerter{
		Client:     client,
		Repository: repo,
		Logger:     l,
	}

	detector := &incident.IncidentDetector{
		KubeClient: kubeClient,
		// TODO: don't hardcode to 1.20
		KubeVersion: incident.KubernetesVersion_1_20,
		Repository:  repo,
		Alerter:     alerter,
		Logger:      l,
	}

	resolver := &incident.IncidentResolver{
		KubeClient: kubeClient,
		// TODO: don't hardcode to 1.20
		KubeVersion: incident.KubernetesVersion_1_20,
		Repository:  repo,
		Alerter:     alerter,
		Logger:      l,
	}

	// trigger resolver through pulsar
	go func() {
		p := pulsar.NewPulsar(1, time.Minute) // pulse every 1 minute

		for range p.Pulsate() {
			err := resolver.Run()

			if err != nil {
				l.Error().Caller().Msgf("resolver exited with error: %v", err)
			}
		}
	}()

	eventController := controllers.EventController{
		KubeClient: kubeClient,
		// TODO: don't hardcode to 1.20
		KubeVersion:      incident.KubernetesVersion_1_20,
		IncidentDetector: detector,
		Repository:       repo,
		LogStore:         logStore,
		Logger:           l,
	}

	go eventController.Start()

	podController := controllers.PodController{
		KubeClient: kubeClient,
		// TODO: don't hardcode to 1.20
		KubeVersion:      incident.KubernetesVersion_1_20,
		IncidentDetector: detector,
		Logger:           l,
	}

	podController.Start()
}

func cleanupEventCache(db *gorm.DB, l *logger.Logger) {
	for {
		l.Info().Caller().Msgf("cleaning up old event caches")

		var olderCache []*models.EventCache

		if err := db.Model(&models.EventCache{}).Where("timestamp <= ?", time.Now().Add(-time.Hour)).Find(&olderCache).Error; err == nil {
			numDeleted := 0

			for _, cache := range olderCache {
				if err := db.Delete(cache).Error; err != nil {
					l.Error().Caller().Msgf("error deleting old event cache with ID: %d. Error: %v\n", cache.ID, err)
					numDeleted++
				}
			}

			l.Info().Caller().Msgf("deleted %d event cache objects from database", numDeleted)
		} else {
			l.Error().Caller().Msgf("error querying for older event cache DB entries: %v\n", err)
		}

		time.Sleep(time.Hour)
	}
}
