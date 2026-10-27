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

package dqd

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const venvPython = "../venv/bin/python"

// skipIfNoDQDCheck skips the test if the project's Python virtualenv
// (with the lark package) or the DQD grammar file (not distributed with
// the project) is not available.
func skipIfNoDQDCheck(t *testing.T) {
	if _, err := os.Stat(venvPython); err != nil {
		t.Skip("venv not available, skipping DQD syntax check")
	}
	if _, err := os.Stat("dqd_grammar.lark"); err != nil {
		t.Skip("DQD grammar not available, skipping DQD syntax check")
	}
}

// checkDQDSyntax validates a DQD query against the reference
// DQD grammar using scripts/dqd_check.py.
func checkDQDSyntax(t *testing.T, query string) {
	cmd := exec.Command(venvPython, "../scripts/dqd_check.py")
	cmd.Stdin = strings.NewReader(query)
	out, err := cmd.CombinedOutput()
	assert.NoError(t, err, string(out))
}
