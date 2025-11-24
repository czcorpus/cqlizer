// Copyright 2025 Tomas Machalek <tomas.machalek@gmail.com>
// Copyright 2025 Department of Linguistics,
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

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/czcorpus/cqlizer/cnf"
	"github.com/czcorpus/cqlizer/dataimport"
	"github.com/czcorpus/cqlizer/eval"
	"github.com/rs/zerolog/log"
	"github.com/vmihailenco/msgpack/v5"
)

func runActionFeaturize(
	ctx context.Context,
	conf *cnf.Conf,
	srcPath, dstPath string,
	debug bool,
) {
	model := eval.NewPredictor(nil, conf)
	dataimport.ReadStatsFile(ctx, srcPath, model)
	model.Deduplicate()

	if debug {
		for i, v := range model.Evaluations {
			fmt.Printf("feats[%d] for %s\n", i, v.OrigQuery)
			fmt.Println(v.Show())
		}

	} else {
		srz, err := msgpack.Marshal(model)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to serialize cql queries features")
			return
		}
		fmt.Println("importing features from ", srcPath)

		file, err := os.Create(dstPath)
		if err != nil {
			log.Fatal().Err(err).Str("file", dstPath).Msg("failed to save features to a file")
			return
		}
		defer file.Close()
		if _, err := file.Write(srz); err != nil {
			log.Fatal().Err(err).Str("file", dstPath).Msg("failed to save features to a file")
			return
		}
	}
}
