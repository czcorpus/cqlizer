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

package apiserver

import (
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/czcorpus/cqlizer/eval/modutils"
	"github.com/gin-gonic/gin"
)

type corpusSelProp struct {
	Name string
	Size int64
}

func (api *apiServer) handleTestPage(ctx *gin.Context) {
	// Build corpus options from configuration
	var corpusOptions strings.Builder
	corpora := make([]corpusSelProp, 0, len(api.conf.CorporaProps))
	for c, v := range api.conf.CorporaProps {
		if v.Size > 100000000 {
			corpora = append(corpora, corpusSelProp{Name: c, Size: int64(v.Size)})
		}
	}
	slices.SortFunc(corpora, func(v1, v2 corpusSelProp) int {
		return int(v2.Size - v1.Size)
	})
	for _, corpus := range corpora {
		corpusOptions.WriteString(
			fmt.Sprintf(
				"<option value=\"%s\">%s (%s)</option>\n",
				corpus.Name, corpus.Name, modutils.FormatRoughSize(corpus.Size),
			),
		)
	}

	// Get URL prefix for proxy support
	urlPrefix := api.conf.TestingPageURLPathPrefix
	if urlPrefix != "" && !strings.HasPrefix(urlPrefix, "/") {
		urlPrefix = "/" + urlPrefix
	}
	urlPrefix = strings.TrimSuffix(urlPrefix, "/")

	slowQueryVoteThreshold := 0.0
	for _, mod := range api.rfEnsemble {
		slowQueryVoteThreshold += mod.threshold
	}
	slowQueryVoteThreshold /= float64(len(api.rfEnsemble))

	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>CQL Query Complexity Predictor - Test Page</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
            min-height: 100vh;
            padding: 20px;
            display: flex;
            justify-content: center;
            align-items: center;
        }

        .container {
            background: white;
            border-radius: 12px;
            box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
            max-width: 800px;
            width: 100%%;
            padding: 40px;
        }

        h1 {
            color: #333;
            margin-bottom: 10px;
            font-size: 28px;
        }

        .form-group {
            margin-bottom: 20px;
        }

        label {
            display: block;
            margin-bottom: 8px;
            color: #555;
            font-weight: 600;
            font-size: 14px;
        }

        select, input[type="text"] {
            width: 100%%;
            padding: 12px;
            border: 2px solid #e0e0e0;
            border-radius: 6px;
            font-size: 14px;
            transition: border-color 0.3s;
        }

        select:focus, input[type="text"]:focus {
            outline: none;
            border-color: #667eea;
        }

        button {
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
            color: white;
            padding: 14px 32px;
            border: none;
            border-radius: 6px;
            font-size: 16px;
            font-weight: 600;
            cursor: pointer;
            transition: transform 0.2s, box-shadow 0.2s;
            width: 100%%;
        }

        button:hover:not(:disabled) {
            transform: translateY(-2px);
            box-shadow: 0 8px 20px rgba(102, 126, 234, 0.4);
        }

        button:active:not(:disabled) {
            transform: translateY(0);
        }

        button:disabled {
            opacity: 0.6;
            cursor: not-allowed;
        }

        .result-box {
            margin-top: 30px;
            padding: 20px;
            border-radius: 8px;
            background: #f8f9fa;
            border: 2px solid #e0e0e0;
            display: none;
        }

        .result-box.show {
            display: block;
        }

        .result-box h2 {
            color: #333;
            margin-bottom: 15px;
            font-size: 20px;
        }

        .result-content {
            background: white;
            padding: 15px;
            border-radius: 6px;
            font-family: 'Courier New', monospace;
            font-size: 13px;
            overflow-x: auto;
            max-height: 400px;
            overflow-y: auto;
        }

        .error {
            color: #d32f2f;
            background: #ffebee;
            border-color: #ef5350;
        }

        .success {
            border-color: #66bb6a;
        }

        .prediction-summary {
            background: #fff3cd;
            border: 2px solid #ffc107;
            padding: 15px;
            border-radius: 6px;
            margin-bottom: 15px;
            font-size: 16px;
            font-weight: 600;
        }

        .prediction-summary.slow {
            background: #ffebee;
            border-color: #f44336;
            color: #c62828;
        }

        .prediction-summary.fast {
            background: #e8f5e9;
            border-color: #4caf50;
            color: #2e7d32;
        }

        .confidence-bar-container {
            margin-top: 15px;
            margin-bottom: 15px;
        }

        .confidence-label {
            font-size: 13px;
            color: #555;
            margin-bottom: 8px;
            display: flex;
            justify-content: space-between;
        }

        .confidence-bar-track {
            width: 100%%;
            height: 24px;
            background: #e0e0e0;
            border-radius: 12px;
            overflow: visible;
            position: relative;
        }

        .threshold-marker {
            position: absolute;
            top: -2px;
            height: 28px;
            width: 3px;
            background: #d32f2f;
            z-index: 10;
            box-shadow: 0 0 4px rgba(211, 47, 47, 0.5);
        }

        .threshold-marker::before {
            content: '';
            position: absolute;
            top: 0;
            left: -2px;
            width: 0;
            height: 0;
            border-left: 4px solid transparent;
            border-right: 4px solid transparent;
            border-top: 6px solid #d32f2f;
        }

        .threshold-marker::after {
            content: '';
            position: absolute;
            bottom: 0;
            left: -2px;
            width: 0;
            height: 0;
            border-left: 4px solid transparent;
            border-right: 4px solid transparent;
            border-bottom: 6px solid #d32f2f;
        }

        .confidence-bar-fill {
            height: 100%%;
            background: linear-gradient(90deg, #4caf50 0%%, #8bc34a 50%%, #ffc107 75%%, #f44336 100%%);
            border-radius: 12px;
            transition: width 0.5s ease;
            display: flex;
            align-items: center;
            justify-content: flex-end;
            padding-right: 8px;
            color: white;
            font-weight: 600;
            font-size: 12px;
            text-shadow: 0 1px 2px rgba(0,0,0,0.3);
        }

        .confidence-bar-fill.low {
            background: #4caf50;
        }

        .confidence-bar-fill.medium {
            background: #ffc107;
        }

        .confidence-bar-fill.high {
            background: #f44336;
        }

        .loading {
            display: inline-block;
            width: 16px;
            height: 16px;
            border: 3px solid rgba(255,255,255,.3);
            border-radius: 50%%;
            border-top-color: #fff;
            animation: spin 1s ease-in-out infinite;
            margin-left: 10px;
        }

        @keyframes spin {
            to { transform: rotate(360deg); }
        }

        .info-text {
            background: #e3f2fd;
            border-left: 4px solid #2196f3;
            padding: 12px;
            margin-bottom: 20px;
            border-radius: 4px;
            font-size: 13px;
            color: #1565c0;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>CQL Query Complexity Predictor</h1>

        <div class="info-text">
            This tool predicts whether a CQL query will be slow or fast based on its complexity.
        </div>

        <form id="cqlForm">
            <div class="form-group">
                <label for="corpus">Corpus:</label>
                <select id="corpus" name="corpus" required>
                    %s
                </select>
            </div>

            <div class="form-group">
                <label for="query">CQL Query:</label>
                <input type="text" id="query" name="query" placeholder='e.g., [lemma=".*"] | [word="test"]' required>
            </div>

            <button type="submit" id="submitBtn">
                Evaluate Query
            </button>
        </form>

        <div class="result-box" id="resultBox">
            <h2>Results</h2>
            <div id="predictionSummary"></div>
            <div class="result-content" id="resultContent"></div>
        </div>
    </div>

    <script>
        const urlPrefix = '%s';
        const slowQueryThreshold = %f;
        const slowQueryThresholdPercent = Math.round(slowQueryThreshold * 100);
        const form = document.getElementById('cqlForm');
        const resultBox = document.getElementById('resultBox');
        const resultContent = document.getElementById('resultContent');
        const predictionSummary = document.getElementById('predictionSummary');
        const submitBtn = document.getElementById('submitBtn');

        form.addEventListener('submit', async (e) => {
            e.preventDefault();

            const corpus = document.getElementById('corpus').value;
            const query = document.getElementById('query').value;

            if (!corpus) {
                alert('Please select a corpus');
                return;
            }

            // Show loading state
            submitBtn.disabled = true;
            submitBtn.innerHTML = 'Evaluating<span class="loading"></span>';
            resultBox.classList.remove('show');

            try {
                const url = urlPrefix + '/cql/' + encodeURIComponent(corpus) + '?q=' + encodeURIComponent(query);
                const response = await fetch(url);
                const data = await response.json();

                // Display results
                resultBox.classList.add('show');
                resultBox.classList.remove('error', 'success');

                if (response.ok) {
                    resultBox.classList.add('success');

                    // Calculate average confidence from votes
                    // Each vote array has [vote_for_false, vote_for_true]
                    // We want the average of vote_for_true (index 1)
                    let totalConfidence = 0;
                    let numModels = 0;

                    if (data.votes && data.votes.length > 0) {
                        data.votes.forEach(vote => {
                            if (vote.votes && vote.votes.length > 1) {
                                totalConfidence += vote.votes[1]; // Index 1 is for "true" (slow query)
                                numModels++;
                            }
                        });
                    }

                    const avgConfidence = numModels > 0 ? totalConfidence / numModels : 0;
                    const confidencePercent = Math.round(avgConfidence * 100);

                    // Create prediction summary
                    const isSlowQuery = data.isSlowQuery;
                    const summaryClass = isSlowQuery ? 'slow' : 'fast';
                    const summaryText = isSlowQuery ? '⚠️ SLOW QUERY PREDICTED' : '✓ FAST QUERY PREDICTED';

                    // Build confidence bar
                    let barClass = 'low';
                    if (avgConfidence >= slowQueryThreshold) {
                        barClass = 'high';
                    } else if (avgConfidence > 0.5) {
                        barClass = 'medium';
                    }

                    predictionSummary.className = 'prediction-summary ' + summaryClass;
                    predictionSummary.innerHTML = summaryText +
                        '<div class="confidence-bar-container">' +
                        '<div class="confidence-label">' +
                        '<span>Slowness factor:</span>' +
                        '<span><strong>' + confidencePercent + '%%</strong></span>' +
                        '</div>' +
                        '<div class="confidence-bar-track">' +
                        '<div class="threshold-marker" style="left: ' + slowQueryThresholdPercent + '%%"></div>' +
                        '<div class="confidence-bar-fill ' + barClass + '" style="width: ' + confidencePercent + '%%">' +
                        (confidencePercent > 15 ? confidencePercent + '%%' : '') +
                        '</div>' +
                        '</div>' +
                        '</div>';

                    // Format JSON nicely
                    resultContent.textContent = JSON.stringify(data, null, 2);
                } else {
                    resultBox.classList.add('error');
                    predictionSummary.className = 'prediction-summary';
                    predictionSummary.textContent = '❌ ERROR';
                    resultContent.textContent = JSON.stringify(data, null, 2);
                }
            } catch (error) {
                resultBox.classList.add('show', 'error');
                predictionSummary.className = 'prediction-summary';
                predictionSummary.textContent = '❌ REQUEST FAILED';
                resultContent.textContent = 'Error: ' + error.message;
            } finally {
                submitBtn.disabled = false;
                submitBtn.innerHTML = 'Evaluate Query';
            }
        });
    </script>
</body>
</html>`, corpusOptions.String(), urlPrefix, slowQueryVoteThreshold)

	ctx.Header("Content-Type", "text/html; charset=utf-8")
	ctx.String(http.StatusOK, html)
}
