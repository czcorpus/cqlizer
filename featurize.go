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
