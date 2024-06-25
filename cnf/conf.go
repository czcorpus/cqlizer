package cnf

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/czcorpus/cnc-gokit/logging"
	"github.com/rs/zerolog/log"
)

const (
	dfltServerWriteTimeoutSecs = 30
	dfltLanguage               = "en"
	dfltMaxNumConcurrentJobs   = 4
	dfltVertMaxNumErrors       = 100
	dfltTimeZone               = "Europe/Prague"
)

type Conf struct {
	srcPath                string
	ListenAddress          string           `json:"listenAddress"`
	PublicURL              string           `json:"publicUrl"`
	ListenPort             int              `json:"listenPort"`
	ServerReadTimeoutSecs  int              `json:"serverReadTimeoutSecs"`
	ServerWriteTimeoutSecs int              `json:"serverWriteTimeoutSecs"`
	CorsAllowedOrigins     []string         `json:"corsAllowedOrigins"`
	TimeZone               string           `json:"timeZone"`
	AuthHeaderName         string           `json:"authHeaderName"`
	AuthTokens             []string         `json:"authTokens"`
	LogFile                string           `json:"logFile"`
	LogLevel               logging.LogLevel `json:"logLevel"`
}

func LoadConfig(path string) *Conf {
	if path == "" {
		log.Fatal().Msg("Cannot load config - path not specified")
	}
	rawData, err := os.ReadFile(path)
	if err != nil {
		log.Fatal().Err(err).Msg("Cannot load config")
	}
	var conf Conf
	conf.srcPath = path
	err = json.Unmarshal(rawData, &conf)
	if err != nil {
		log.Fatal().Err(err).Msg("Cannot load config")
	}
	return &conf
}

func ValidateAndDefaults(conf *Conf) {
	if conf.ServerWriteTimeoutSecs == 0 {
		conf.ServerWriteTimeoutSecs = dfltServerWriteTimeoutSecs
		log.Warn().Msgf(
			"serverWriteTimeoutSecs not specified, using default: %d",
			dfltServerWriteTimeoutSecs,
		)
	}
	if conf.PublicURL == "" {
		conf.PublicURL = fmt.Sprintf("http://%s", conf.ListenAddress)
		log.Warn().Str("address", conf.PublicURL).Msg("publicUrl not set, using listenAddress")
	}

	if conf.TimeZone == "" {
		log.Warn().
			Str("timeZone", dfltTimeZone).
			Msg("time zone not specified, using default")
	}
	if _, err := time.LoadLocation(conf.TimeZone); err != nil {
		log.Fatal().Err(err).Msg("invalid time zone")
	}
}
