package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/czcorpus/cqlizer/cql"
)

type jsonlQueryRecord struct {
	Query   string   `json:"query"`
	Freq    int64    `json:"freq"`
	Corpora []string `json:"corpora"`
}

type fingerprintGroupEntry struct {
	queries []string
	freq    int64
}

// GetQueriesFileFingerprintsFromJSONL
// expected record:
// {"query": "[a-z]+,<QUERY>", freq: number, corpora: []string}
// It behaves like GetQueriesFileFingerprints but when groupItems is set,
// it also prints the total frequency of all the grouped items as the
// last (tab-separated) column.
func GetQueriesFileFingerprintsFromJSONL(queriesFilePath string, groupItems bool, examplesPerGroup int) {
	f, err := os.Open(queriesFilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open queries file: %s\n", err)
		os.Exit(1)
	}
	defer f.Close()

	reader := bufio.NewReader(f)
	grouped := make(map[string]fingerprintGroupEntry)
	for {
		line, err := reader.ReadString('\n')
		line = strings.TrimRight(line, "\r\n")
		if line != "" {
			var rec jsonlQueryRecord
			if jsonErr := json.Unmarshal([]byte(line), &rec); jsonErr != nil {
				fmt.Fprintf(os.Stderr, "failed to parse JSONL record %q: %s\n", line, jsonErr)

			} else {
				query := rec.Query
				if strings.HasPrefix(query, "q") {
					query = query[1:]

				} else if idx := strings.Index(query, ","); idx >= 0 {
					query = query[idx+1:]
				}
				fingerPrint, fpErr := GetQueryTypeFingerprint(query)
				if fpErr != nil {
					fmt.Fprintf(os.Stderr, "failed to get fingerprint for query %q: %s\n", query, fpErr)

				} else if groupItems {
					curr, ok := grouped[fingerPrint]
					if !ok {
						curr = fingerprintGroupEntry{queries: make([]string, 0, 4)}
					}
					curr.queries = append(curr.queries, query)
					curr.freq += rec.Freq
					grouped[fingerPrint] = curr

				} else {
					fmt.Printf("%s\t%s\n", query, fingerPrint)
				}
			}
		}
		if err != nil {
			if !errors.Is(err, io.EOF) {
				fmt.Fprintf(os.Stderr, "failed to read queries file: %s\n", err)
				os.Exit(1)
			}
			break
		}
	}
	if groupItems {
		tmp := make([]fingerprintGroupEntry, len(grouped))
		i := 0
		for _, v := range grouped {
			tmp[i] = v
			i++
		}
		slices.SortFunc(tmp, func(s1, s2 fingerprintGroupEntry) int {
			return int(s2.freq) - int(s1.freq)
		})
		for _, v := range tmp {
			for i := range examplesPerGroup {
				if i >= len(v.queries) {
					break
				}
				if i == 0 {
					fmt.Printf("%s\t%d\n", v.queries[i], v.freq)

				} else {
					fmt.Printf("%s\t--\n", v.queries[i])
				}
			}
		}
	}
}

func GetQueriesFileFingerprints(queriesFilePath string, groupItems bool, examplesPerGroup int) {
	// the entry looks like this [a-z]+,<QUERY>
	// so there is a prefix we must remove
	f, err := os.Open(queriesFilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open queries file: %s\n", err)
		os.Exit(1)
	}
	defer f.Close()

	reader := bufio.NewReader(f)
	grouped := make(map[string]fingerprintGroupEntry)
	for {
		line, err := reader.ReadString('\n')
		line = strings.TrimRight(line, "\r\n")
		if line != "" {
			query := line
			if strings.HasPrefix(line, "q") {
				query = line[1:]

			} else if idx := strings.Index(line, ","); idx >= 0 {
				query = line[idx+1:]
			}
			fingerPrint, fpErr := GetQueryTypeFingerprint(query)
			if fpErr != nil {
				fmt.Fprintf(os.Stderr, "failed to get fingerprint for query %q: %s\n", query, fpErr)

			} else if groupItems {
				curr, ok := grouped[fingerPrint]
				if !ok {
					curr = fingerprintGroupEntry{queries: make([]string, 0, 4)}
				}
				curr.queries = append(curr.queries, query)
				grouped[fingerPrint] = curr

			} else {
				fmt.Printf("%s\t%s\n", query, fingerPrint)
			}
		}
		if err != nil {
			if !errors.Is(err, io.EOF) {
				fmt.Fprintf(os.Stderr, "failed to read queries file: %s\n", err)
				os.Exit(1)
			}
			break
		}
	}
	if groupItems {
		tmp := make([]fingerprintGroupEntry, len(grouped))
		i := 0
		for _, v := range grouped {
			slices.SortFunc(v.queries, func(s1, s2 string) int {
				return len(s1) - len(s2)
			})
			tmp[i] = v
			i++
		}
		slices.SortFunc(tmp, func(s1, s2 fingerprintGroupEntry) int {
			return len(s1.queries[0]) - len(s2.queries[0])
		})
		for _, v := range tmp {
			for i := range examplesPerGroup {
				if i >= len(v.queries) {
					break
				}
				fmt.Println(v.queries[i])
			}
		}
	}
}

func GetQueryTypeFingerprint(q string) (string, error) {
	query, err := cql.ParseCQL("", q)
	if err != nil {
		return "", fmt.Errorf("failed to parse query: %w", err)
	}
	var fingerPrint strings.Builder
	query.DFS(func(v cql.ASTNode) bool { return true }, func(v cql.ASTNode) {
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
