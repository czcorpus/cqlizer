package main

import (
	"fmt"
	"os"

	"github.com/czcorpus/cnc-gokit/collections"
	"github.com/czcorpus/cqlizer/benchmark"
	"github.com/czcorpus/cqlizer/cnf"
	"github.com/czcorpus/cqlizer/logproc"
	"github.com/czcorpus/cqlizer/prediction"
	"github.com/czcorpus/cqlizer/stats"
	"github.com/fatih/color"
	"github.com/rs/zerolog/log"
)

const (
	errColor = color.FgHiRed
)

func runKontextImport(conf *cnf.Conf, path string, addToTrainingSet bool) {
	err := logproc.ImportLog(conf, path, addToTrainingSet)
	if err != nil {
		color.New(errColor).Fprintln(os.Stderr, err)
	}
}

func runSizesImport(conf *cnf.Conf, path string) {
	db, err := stats.NewDatabase(conf.WorkingDBPath)
	if err != nil {
		color.New(errColor).Fprintln(os.Stderr, err)
	}
	err = db.Init()
	if err != nil {
		color.New(errColor).Fprintln(os.Stderr, err)
	}
	err = db.ImportCorpusSizesFromCSV(path)
	if err != nil {
		color.New(errColor).Fprintln(os.Stderr, err)
	}
}

func runBenchmark(conf *cnf.Conf, overwriteBenchmarked bool) {
	db, err := stats.NewDatabase(conf.WorkingDBPath)
	if err != nil {
		color.New(errColor).Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	err = db.Init()
	if err != nil {
		color.New(errColor).Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	exe := benchmark.NewExecutor(
		conf,
		db,
	)
	err = exe.RunFull(overwriteBenchmarked)
	if err != nil {
		color.New(errColor).Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runLearning(conf *cnf.Conf, threshold float64) {
	db, err := stats.NewDatabase(conf.WorkingDBPath)
	if err != nil {
		color.New(errColor).Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	err = db.Init()
	if err != nil {
		color.New(errColor).Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	eng := prediction.NewEngine(conf, db)
	err = eng.Train(threshold)
	if err != nil {
		color.New(errColor).Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runTrainingReplay(conf *cnf.Conf, trainingID int) {
	statsDB, err := stats.NewDatabase(conf.WorkingDBPath)
	if err != nil {
		color.New(errColor).Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	err = statsDB.Init()
	if err != nil {
		color.New(errColor).Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if trainingID == 0 {
		log.Warn().Msg("no training ID provided, going to use the latest one")
		trainingID, err = statsDB.GetLatestTrainingID()
		if err != nil {
			color.New(errColor).Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	threshold, err := statsDB.GetTrainingThreshold(trainingID)
	if err != nil {
		color.New(errColor).Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	tdata, err := statsDB.GetTrainingData(trainingID)
	if err != nil {
		color.New(errColor).Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	vdata, err := statsDB.GetTrainingValidationData(trainingID)
	if err != nil {
		color.New(errColor).Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	eng := prediction.NewEngine(conf, statsDB)
	_, err = eng.TrainReplay(threshold, tdata, vdata)
	if err != nil {
		color.New(errColor).Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runEvaluation(conf *cnf.Conf, trainingID, numSamples, sampleSize int) {
	statsDB, err := stats.NewDatabase(conf.WorkingDBPath)
	if err != nil {
		log.Error().Err(err).Msg("failed to run evaluation")
		os.Exit(1)
		return
	}

	err = statsDB.Init()
	if err != nil {
		color.New(errColor).Fprintln(os.Stderr, err)
		return
	}

	if trainingID == 0 {
		log.Warn().Msg("no training ID provided, going to use the latest one")
		trainingID, err = statsDB.GetLatestTrainingID()
		if err != nil {
			color.New(errColor).Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	model, threshold, err := loadModel(conf, statsDB, trainingID)
	if err != nil {
		color.New(errColor).Fprintln(os.Stderr, err)
		os.Exit(1)
		return
	}

	recs, err := statsDB.GetAllRecords(
		stats.ListFilter{}.SetBenchmarked(true).SetTrainingExcluded(true))
	if err != nil {
		color.New(errColor).Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if len(recs) == 0 {
		err := fmt.Errorf("no validation data found")
		color.New(errColor).Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	log.Info().
		Int("numItems", len(recs)).
		Msg("fetched items for sampling")

	var avgPrecision, avgRecall float64
	for i := 0; i < numSamples; i++ {
		smpl := collections.SliceSample(recs, sampleSize)
		log.Debug().Int("sampleNum", i).Msg("going to evaluate next sample")
		eng := prediction.NewEngine(conf, statsDB)
		result, err := eng.Evaluate(model, smpl, threshold, func(itemID string, prediction bool) error {
			return nil
		})
		if err != nil {
			log.Error().Err(err).Msg("failed to run validation, exiting")
			os.Exit(1)
		}
		log.Info().
			Int("sampleNum", i).
			Int("truePositives", result.TruePositives).
			Int("falsePositives", result.FalsePositives).
			Int("falseNegatives", result.FalseNegatives).
			Int("total", result.TotalTests).
			Float64("precision", result.Precision()).
			Float64("recall", result.Recall()).
			Send()
		avgPrecision += result.Precision()
		avgRecall += result.Recall()
	}
	fmt.Println("----------------------------------------------------")
	fmt.Println("sample size: ", sampleSize)
	fmt.Println("number of samples (test runs): ", numSamples)
	fmt.Printf("AVG PRECISION: %01.2f\n", avgPrecision/float64(numSamples))
	fmt.Printf("AVG RECALL: %01.2f\n", avgRecall/float64(numSamples))
	fmt.Println("----------------------------------------------------")
}
