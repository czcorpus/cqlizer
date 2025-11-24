// Copyright 2024 Tomas Machalek <tomas.machalek@gmail.com>
// Copyright 2024 Department of Linguistics,
// Faculty of Arts, Charles University
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cnf

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/czcorpus/cnc-gokit/logging"
	"github.com/czcorpus/cqlizer/eval/feats"
	"github.com/rs/zerolog/log"
)

const (
	dfltServerWriteTimeoutSecs = 30
	dfltLanguage               = "en"
	dfltMaxNumConcurrentJobs   = 4
	dfltVertMaxNumErrors       = 100
	dfltTimeZone               = "Europe/Prague"
)

type RFEnsembleConf struct {
	ModelPath     string  `json:"modelPath"`
	VoteThreshold float64 `json:"voteThreshold"`
	ModelType     string  `json:"modelType"`
	Disabled      bool    `json:"disabled"`
}

type Conf struct {
	srcPath                  string
	Logging                  logging.LoggingConf          `json:"logging"`
	ListenAddress            string                       `json:"listenAddress"`
	PublicURL                string                       `json:"publicUrl"`
	ListenPort               int                          `json:"listenPort"`
	ServerReadTimeoutSecs    int                          `json:"serverReadTimeoutSecs"`
	TestingPageURLPathPrefix string                       `json:"testingPageURLPathPrefix"`
	ServerWriteTimeoutSecs   int                          `json:"serverWriteTimeoutSecs"`
	CorsAllowedOrigins       []string                     `json:"corsAllowedOrigins"`
	TimeZone                 string                       `json:"timeZone"`
	RFEnsemble               []RFEnsembleConf             `json:"rfEnsemble"`
	CorporaProps             map[string]feats.CorpusProps `json:"corporaProps"`

	// SyntheticTimeCorrection - for stats records generated via benchmarking,
	// it may be needed to increase the times as MQuery will probably perform a bit better
	// and if performed during low traffic hours, this difference can be even bigger.
	SyntheticTimeCorrection float64 `json:"syntheticTimeCorrection"`
	MQueryBenchmarkingURL   string  `json:"mqueryBenchmarkingUrl"`
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

	if conf.SyntheticTimeCorrection == 0 {
		log.Warn().Msg("SyntheticRecordsTimeCorrection is not set - we must set it to 1")
		conf.SyntheticTimeCorrection = 1
	}
}
