package envconf

import (
	"github.com/porter-dev/porter-agent/pkg/httpclient"
	"github.com/porter-dev/porter/api/server/shared/config/env"
)

type LogStoreConf struct {
	LogStoreAddress string `env:"LOG_STORE_ADDRESS,default=:9096"`
	LogStoreKind    string `env:"LOG_STORE_KIND,default=memory"`
}
type EnvDecoderConf struct {
	Debug     bool   `env:"DEBUG,default=true"`
	SentryDSN string `env:"SENTRY_DSN"`
	SentryEnv string `env:"SENTRY_ENV,default=dev"`

	LogStoreConf   LogStoreConf
	HTTPClientConf httpclient.HTTPClientConf
	DBConf         env.DBConf
}
