package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/czcorpus/cqlizer/cql"
)

func GetQueriesFileFingerprints(queriesFilePath string) {
	// the entry looks like this [a-z]+,<QUERY>
	// so there is a prefix we must remove
	f, err := os.Open(queriesFilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open queries file: %s\n", err)
		os.Exit(1)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		query := line
		if idx := strings.Index(line, ","); idx >= 0 {
			query = line[idx+1:]
		}
		fingerPrint, err := GetQueryTypeFingerprint(query)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to get fingerprint for query %q: %s\n", query, err)
			continue
		}
		fmt.Printf("%s\t%s\n", query, fingerPrint)
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to read queries file: %s\n", err)
		os.Exit(1)
	}
}

func GetQueryTypeFingerprint(q string) (string, error) {
	query, err := cql.ParseCQL("", q)
	if err != nil {
		return "", fmt.Errorf("failed to parse query: %w", err)
	}
	var fingerPrint strings.Builder
	query.DFS(func(v cql.ASTNode) {
		switch v.(type) {
		case *cql.Sequence:
			fingerPrint.WriteString(";Sequence")
		case *cql.Seq:
			fingerPrint.WriteString(";Seq")
		case *cql.GlobPart:
			fingerPrint.WriteString(";GlobPart")
		case *cql.WithinOrContaining:
			fingerPrint.WriteString(";WithinOrContaining")
		case *cql.GlobCond:
			fingerPrint.WriteString(";GlobCond")
		case *cql.Structure:
			fingerPrint.WriteString(";Structure")
		case *cql.AttValList:
			fingerPrint.WriteString(";AttValList")
		case *cql.NumberedPosition:
			fingerPrint.WriteString(";NumberedPosition")
		case *cql.OnePosition:
			fingerPrint.WriteString(";OnePosition")
		case *cql.Position:
			fingerPrint.WriteString(";Position")
		case *cql.RegExp:
			fingerPrint.WriteString(";RegExp")
		case *cql.MuPart:
			fingerPrint.WriteString(";MuPart")
		case *cql.UnionOp:
			fingerPrint.WriteString(";UnionOp")
		case *cql.MeetOp:
			fingerPrint.WriteString(";MeetOp")
		case *cql.Repetition:
			fingerPrint.WriteString(";Repetition")
		case *cql.AtomQuery:
			fingerPrint.WriteString(";AtomQuery")
		case *cql.RepOpt:
			fingerPrint.WriteString(";RepOpt")
		case *cql.OpenStructTag:
			fingerPrint.WriteString(";OpenStructTag")
		case *cql.CloseStructTag:
			fingerPrint.WriteString(";CloseStructTag")
		case *cql.AlignedPart:
			fingerPrint.WriteString(";AlignedPart")
		case *cql.AttValAnd:
			fingerPrint.WriteString(";AttValAnd")
		case *cql.AttVal:
			fingerPrint.WriteString(";AttVal")
		case *cql.WithinNumber:
			fingerPrint.WriteString(";WithinNumber")
		case *cql.RegExpRaw:
			fingerPrint.WriteString(";RegExpRaw")
		case *cql.RawString:
			fingerPrint.WriteString(";RawString")
		case *cql.SimpleString:
			fingerPrint.WriteString(";SimpleString")
		}
	})
	return fingerPrint.String(), nil
}
