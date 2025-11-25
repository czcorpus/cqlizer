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
	"io"
	"math"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/czcorpus/cqlizer/cnf"
	"github.com/czcorpus/cqlizer/eval"
	"github.com/czcorpus/cqlizer/eval/nn"
	"github.com/czcorpus/cqlizer/eval/rf"
	"github.com/czcorpus/cqlizer/eval/xg"
	"github.com/rs/zerolog/log"
	"github.com/schollz/progressbar/v3"
	"github.com/vmihailenco/msgpack/v5"
)

func runActionKlogImport(
	conf *cnf.Conf,
	srcPath string,
	modelType string,
	numTrees int,
	voteThreshold float64,
	misclassLogPath string,
) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	/*
		model := &eval.BasicModel{
			SlowQueryPercentile: slowQueryPerc,
		}
		dataimport.ReadStatsFile(ctx, srcPath, model)
	*/

	f, err := os.Open(srcPath)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to open features file")
		return
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to open features file")
		return
	}

	var mlModel eval.MLModel
	switch modelType {
	case "rf":
		mlModel = rf.NewModel(numTrees, voteThreshold)
	case "nn":
		mlModel = nn.NewModel()
	case "xg":
		mlModel = xg.NewModel()
	default:
		log.Fatal().Str("modelType", modelType).Msg("Unknown model")
		return
	}

	model := eval.NewPredictor(mlModel, conf)
	if err := msgpack.Unmarshal(data, &model); err != nil {
		log.Fatal().Err(err).Msg("failed to open features file")
		return
	}

	allEvals := model.BalanceSample()
	reporter := &eval.Reporter{
		RFAccuracyScript:       rfChartScript,
		MisclassQueriesOutPath: misclassLogPath,
	}

	if err := model.CreateAndTestModel(ctx, allEvals, srcPath, reporter); err != nil {
		fmt.Fprintf(os.Stderr, "RF training failed: %v\n", err)
		os.Exit(1)
	}
}

func runActionEvaluate(
	conf *cnf.Conf,
	modelPath string,
	modelType string,
	tstDataPath string,
	misclassLogPath string,
) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	mlModel, err := eval.GetMLModel(modelType, modelPath)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load the ML model")
		return
	}

	f, err := os.Open(tstDataPath)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to open features file")
		return
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to open features file")
		return
	}

	predictor := eval.NewPredictor(mlModel, conf)
	if err := msgpack.Unmarshal(data, &predictor); err != nil {
		log.Fatal().Err(err).Msg("failed to open features file")
		return
	}
	predictor.FindAndSetDataMidpoint()

	reporter := &eval.Reporter{
		RFAccuracyScript:       rfChartScript,
		MisclassQueriesOutPath: misclassLogPath,
	}

	log.Info().
		Int("evalDataSize", len(predictor.Evaluations)).
		Msg("calculating precision and recall using full data")

	bar := progressbar.Default(int64(math.Ceil((1-0.5)/0.01)), "testing the model")
	var csv strings.Builder
	csv.WriteString("vote;precision;recall;f-beta\n")
	for v := 0.5; v < 1; v += 0.01 {
		select {
		case <-ctx.Done():
			return
		default:
		}
		mlModel.SetClassThreshold(v)
		precall := predictor.PrecisionAndRecall(reporter)
		csv.WriteString(precall.CSV(v) + "\n")
		bar.Add(1)
	}
	unixt := time.Now().Unix()
	chartPath := fmt.Sprintf("./test-%d.png", unixt)
	if err := reporter.PlotRFAccuracy(csv.String(), mlModel.GetInfo(), chartPath); err != nil {
		log.Fatal().Err(err).Msgf("failed to generate accuracy chart")
		return

	} else {
		log.Info().Str("file", chartPath).Msg("saved evaluation chart")
	}
	reporter.SaveMisclassifiedQueries()
}
