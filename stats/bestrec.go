// Copyright 2024 Tomas Machalek <tomas.machalek@gmail.com>
// Copyright 2024 Institute of the Czech National Corpus,
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

package stats

import (
	"fmt"
)

type MatchItem struct {
	Record   *DBRecord
	Distance int
}

type BestMatches struct {
	size int
	data []MatchItem
}

func (bm *BestMatches) AvgBenchTime() float64 {
	var ans float64
	for _, v := range bm.data {
		ans += v.Record.BenchTime
	}
	return ans / float64(len(bm.data))
}

func (bm *BestMatches) SmartBenchTime() float64 {
	bestItemDist := bm.data[0].Distance
	var numIncluded float64
	var avg float64
	for i := 1; i < len(bm.data); i++ {
		if bm.data[i].Distance == bestItemDist {
			avg += bm.data[i].Record.BenchTime
			numIncluded++
		}
	}
	return avg / numIncluded
}

func (bm *BestMatches) WorstBenchTime() float64 {
	var ans float64
	for _, v := range bm.data {
		if v.Record.BenchTime > ans {
			ans = v.Record.BenchTime
		}
	}
	return ans
}

func (bm *BestMatches) Print() {
	for _, v := range bm.data {
		fmt.Printf("%s (norm: %s, time: %01.2f)\n", v.Record.Query, v.Record.QueryNormalized, v.Record.BenchTime)
	}
}

func (bm *BestMatches) At(idx int) MatchItem {
	return bm.data[idx]
}

func (bm *BestMatches) Items() []MatchItem {
	return bm.data
}

func (bm *BestMatches) TryAdd(rec *DBRecord, dist int) bool {
	pos := -1
	for i := 0; i < len(bm.data); i++ {
		if dist < bm.data[i].Distance {
			pos = i
			break
		}
	}
	if pos == -1 && len(bm.data) < bm.size {
		bm.data = append(
			bm.data,
			MatchItem{
				Record:   rec,
				Distance: dist,
			},
		)
		pos = len(bm.data) - 1

	} else if pos >= 0 {
		tmp := make([]MatchItem, len(bm.data[pos:]))
		copy(tmp, bm.data[pos:])
		bm.data = bm.data[:pos]
		bm.data = append(
			bm.data,
			MatchItem{
				Record:   rec,
				Distance: dist,
			},
		)
		bm.data = append(bm.data, tmp...)
	}
	if len(bm.data) > bm.size {
		bm.data = bm.data[:bm.size]
	}
	return pos > -1
}

func NewBestMatches(size int) *BestMatches {
	return &BestMatches{
		size: size,
		data: make([]MatchItem, 0, size+1),
	}
}
