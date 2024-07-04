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
	"crypto/sha1"
	"encoding/hex"
	"time"
)

func IdempotentID(created time.Time, query string) string {
	sum := sha1.New()
	_, err := sum.Write([]byte(created.String() + "#"))
	if err != nil {
		panic("problem generating hash")
	}
	_, err = sum.Write([]byte(query))
	if err != nil {
		panic("problem generating hash")
	}

	return hex.EncodeToString(sum.Sum(nil))
}
