package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/czcorpus/cqlizer/dqd"
)

// runActionCQL2DQD translates a CQL query to DQD and prints the result
// to stdout. If query is empty, it is read from stdin. Corpus props are
// created from LCP layer info (layerInfoPath) or loaded from a props file
// (propsPath). If none is specified, default corpus props are used.
func runActionCQL2DQD(propsPath, layerInfoPath, query string) {
	cp, err := loadCorpusProps(propsPath, layerInfoPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
	if query == "" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to read query from stdin: %s\n", err)
			os.Exit(1)
		}
		query = strings.TrimSpace(string(data))
	}
	ans, err := dqd.CQL2DQD(cp, dqd.ViewOptions{}, query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
	fmt.Print(ans)
}

func loadCorpusProps(propsPath, layerInfoPath string) (dqd.CorpusProps, error) {
	switch {
	case propsPath != "" && layerInfoPath != "":
		return dqd.CorpusProps{}, errors.New("-corpus-props and -layer-info cannot be used together")
	case propsPath != "":
		return dqd.LoadCorpusProps(propsPath)
	case layerInfoPath != "":
		li, err := dqd.LoadLayerInfo(layerInfoPath)
		if err != nil {
			return dqd.CorpusProps{}, err
		}
		return dqd.CorpusPropsFromLayerInfo(li)
	}
	return dqd.DefaultCorpusProps(), nil
}
