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

package modutils

import (
	"fmt"
	"regexp"
)

var feat2modelRegexp = regexp.MustCompile(`(.+\.v\d+\.\d+).*`)

func FormatRoughSize(value int64) string {
	if value < 100000 {
		return "~0"
	}

	if value >= 1000000000 { // 1 billion or more
		billions := float64(value) / 1000000000.0
		return fmt.Sprintf("%.1fG", billions)
	}

	if value >= 100000 { // 1 million or more
		millions := float64(value) / 1000000.0
		return fmt.Sprintf("%.1fM", millions)
	}

	// Between 100,000 and 1,000,000
	return fmt.Sprintf("%d", value)
}

func ExtractModelNameBaseFromFeatFile(filename string) string {
	return feat2modelRegexp.ReplaceAllString(filename, "$1")
}
