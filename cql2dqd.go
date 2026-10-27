package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/czcorpus/cqlizer/dqd"
)

// runActionCQL2DQD translates a CQL query to DQD and prints the result
// to stdout. If query is empty, it is read from stdin. If propsPath is
// empty, default corpus props are used.
func runActionCQL2DQD(propsPath, query string) {
	cp := dqd.DefaultCorpusProps()
	if propsPath != "" {
		var err error
		cp, err = dqd.LoadCorpusProps(propsPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(1)
		}
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
