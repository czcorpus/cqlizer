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

// This file provides a test page for the Natural Language to CQL translation feature.
// The test page allows users to:
// - Edit the system prompt used for translation
// - Enter natural language queries and translate them to CQL
// - Save modified system prompts with automatic versioning (timestamp-based)
// - Load the current system prompt from the translator
//
// The system prompt versions are saved to the "prompts/" directory with filenames
// in the format: system_prompt_YYYYMMDD-HHMMSS.txt
//
// Access the test page at: /test-nl

package apiserver

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func (api *apiServer) handleNLTestPage(ctx *gin.Context) {
	// Get URL prefix for proxy support
	urlPrefix := api.conf.TestingPageURLPathPrefix
	if urlPrefix != "" && !strings.HasPrefix(urlPrefix, "/") {
		urlPrefix = "/" + urlPrefix
	}
	urlPrefix = strings.TrimSuffix(urlPrefix, "/")

	// Get initial system prompt from the CQL translator
	initialSystemPrompt := api.cqlTranslator.GetSystemPrompt()
	modelName := api.cqlTranslator.GetModelName()

	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Natural Language to CQL - Test Page</title>
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
            flex-direction: column;
            justify-content: center;
            align-items: center;
        }

        .container {
            background: white;
            border-radius: 12px;
            box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
            max-width: 900px;
            width: 100%%;
            padding: 40px;
        }

        h1 {
            color: #333;
            margin-bottom: 10px;
            font-size: 28px;
        }

        label {
            display: block;
            margin-bottom: 8px;
            color: #555;
            font-weight: 600;
            font-size: 14px;
        }

        textarea {
            width: 100%%;
            padding: 12px;
            border: 2px solid #e0e0e0;
            border-radius: 6px;
            font-size: 14px;
            font-family: 'Courier New', monospace;
            transition: border-color 0.3s;
            resize: vertical;
        }

        textarea:focus {
            outline: none;
            border-color: #667eea;
        }

        .button-group {
            display: flex;
            gap: 10px;
            margin-top: 10px;
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
            flex: 1;
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

        .secondary-btn {
            background: linear-gradient(135deg, #48bb78 0%%, #38a169 100%%);
        }

        .secondary-btn:hover:not(:disabled) {
            box-shadow: 0 8px 20px rgba(72, 187, 120, 0.4);
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
            max-height: 300px;
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

        .info-text p {
        	margin: 0.5rem 0 0.5rem 0;
        }

        .version-info {
            padding: 10px 12px;
            margin-bottom: 20px;
            font-size: 11px;
            color: #dedede;
            font-family: 'Courier New', monospace;
        }

        .version-info .version-label {
            font-weight: 600;
            color: #bbb;
            margin-right: 8px;
        }

        .version-info .version-item {
            display: inline-block;
            margin-right: 15px;
        }

        .version-info .version-item:last-child {
            margin-right: 0;
        }

        .save-status {
            margin-top: 10px;
            padding: 10px;
            border-radius: 4px;
            font-size: 13px;
            display: none;
        }

        .save-status.show {
            display: block;
        }

        .save-status.success-status {
            background: #e8f5e9;
            border: 1px solid #4caf50;
            color: #2e7d32;
        }

        .save-status.error-status {
            background: #ffebee;
            border: 1px solid #f44336;
            color: #c62828;
        }

        #systemPrompt {
            min-height: 350px;
            font-size: 12px;
            background-color: #ededed;
        }

        #userInput {
            min-height: 80px;
        }

        select {
            width: 100%%;
            padding: 12px;
            border: 2px solid #e0e0e0;
            border-radius: 6px;
            font-size: 14px;
            background: white;
            cursor: pointer;
            transition: border-color 0.3s;
        }

        select:focus {
            outline: none;
            border-color: #667eea;
        }

        input[type="text"] {
            width: 100%%;
            padding: 12px;
            border: 2px solid #e0e0e0;
            border-radius: 6px;
            font-size: 14px;
            transition: border-color 0.3s;
        }

        input[type="text"]:focus {
            outline: none;
            border-color: #667eea;
        }

        .prompt-fieldset {
        	margin-top: 1rem;
        }

        .prompt-fieldset .controls {
        	display: flex;
        	align-items: center;
        }

        .prompt-fieldset .controls select {
        	max-width: 50%%;
         	margin-right: 1rem;
        }

        .save-controls {
            display: flex;
            gap: 10px;
            align-items: flex-end;
            margin-top: 1rem;
        }

        .save-controls .form-group {
            flex: 1;
            margin-bottom: 0;
        }

        .query-fieldset {
        	margin-top: 1em;
         	margin-bottom: 1em;
        }

        .save-controls button {
            flex-shrink: 0;
        }

        /* Modal styles */
        .modal {
            display: none;
            position: fixed;
            z-index: 1000;
            left: 0;
            top: 0;
            width: 100%%;
            height: 100%%;
            background-color: rgba(0, 0, 0, 0.5);
        }

        .modal-content {
            background-color: white;
            margin: 5%% auto;
            padding: 20px;
            border-radius: 12px;
            max-width: 800px;
            max-height: 80%%;
            overflow-y: auto;
            box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
        }

        .modal-content h2 {
            margin-bottom: 15px;
            color: #333;
        }

        .modal-close {
            float: right;
            font-size: 28px;
            font-weight: bold;
            color: #aaa;
            cursor: pointer;
        }

        .modal-close:hover {
            color: #333;
        }

        .tools-json {
            background: #f5f5f5;
            padding: 15px;
            border-radius: 6px;
            font-family: 'Courier New', monospace;
            font-size: 12px;
            white-space: pre-wrap;
            word-wrap: break-word;
            max-height: 500px;
            overflow-y: auto;
        }

        #showToolsLink {
            color: #1565c0;
            text-decoration: underline;
            cursor: pointer;
        }

        #showToolsLink:hover {
            color: #0d47a1;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>CQLizer/NL2CQL - an Orchestration Logic Test</h1>

        <div class="info-text">
        	<p>
            This tool translates natural language queries into CQL (Corpus Query Language).
            You can edit the system prompt to customize the translation behavior.
            </p>
            <p>
            Used LLM: <strong>%s</strong> (<a href="#" id="showToolsLink">show defined tools</a>)<br>
            The default system prompt was created with significant assistance from Claude Sonnet 4.5.
            </p>
        </div>

        <!-- Tools Modal -->
        <div id="toolsModal" class="modal">
            <div class="modal-content">
                <span class="modal-close">&times;</span>
                <h2>Defined LLM Tools</h2>
                <pre id="toolsContent" class="tools-json"></pre>
            </div>
        </div>

        <form id="nlForm">
            <div class="form-group">
                <label for="systemPrompt">System Prompt (editable):</label>
                <textarea id="systemPrompt" name="systemPrompt">%s</textarea>

                <div class="prompt-fieldset">
                	<label for="promptSelect">Load from saved prompts:</label>
                 	<div class="controls">
                        <select id="promptSelect">
                            <option value="">-- Select a prompt file --</option>
                        </select>
                        <button type="button" class="secondary-btn" id="loadDefaultBtn" style="width: 100%%;">
                            Load Default (sysprompt.txt)
                        </button>
					</div>
                </div>

                <div class="save-controls">
                    <div class="form-group">
                        <label for="promptName">Custom name (optional):</label>
                        <input type="text" id="promptName" placeholder="e.g., experiment-1">
                    </div>
                    <button type="button" class="secondary-btn" id="savePromptBtn">
                        Save System Prompt
                    </button>
                </div>
                <div class="save-status" id="saveStatus"></div>
            </div>

            <div class="query-fieldset">
                <label for="userInput">Natural Language Query:</label>
                <textarea id="userInput" name="userInput" placeholder='e.g., Find all occurrences of the word "test" in the corpus' required></textarea>
            </div>

            <button type="submit" id="submitBtn">
                Translate to CQL
            </button>
        </form>

        <div class="result-box" id="resultBox">
            <h2>Results</h2>
            <pre class="result-content" id="resultContent"></pre>
        </div>
    </div>

    <footer class="version-info">
        <span class="version-item"><strong class="version-label">Version:</strong>%s</span>
        <span class="version-item"><strong class="version-label">Build:</strong>%s</span>
    </footer>

    <script>
        const urlPrefix = '%s';
        const form = document.getElementById('nlForm');
        const resultBox = document.getElementById('resultBox');
        const resultContent = document.getElementById('resultContent');
        const submitBtn = document.getElementById('submitBtn');
        const savePromptBtn = document.getElementById('savePromptBtn');
        const loadDefaultBtn = document.getElementById('loadDefaultBtn');
        const systemPromptTextarea = document.getElementById('systemPrompt');
        const saveStatus = document.getElementById('saveStatus');
        const promptSelect = document.getElementById('promptSelect');
        const promptName = document.getElementById('promptName');
        const toolsModal = document.getElementById('toolsModal');
        const toolsContent = document.getElementById('toolsContent');
        const showToolsLink = document.getElementById('showToolsLink');
        const modalClose = document.querySelector('.modal-close');

        // Handle show tools link
        showToolsLink.addEventListener('click', async (e) => {
            e.preventDefault();
            toolsContent.textContent = 'Loading...';
            toolsModal.style.display = 'block';

            try {
                const url = urlPrefix + '/nl-to-cql/tools';
                const response = await fetch(url);
                if (response.ok) {
                    const data = await response.json();
                    toolsContent.textContent = data.tools;
                } else {
                    toolsContent.textContent = 'Error loading tools';
                }
            } catch (error) {
                toolsContent.textContent = 'Error: ' + error.message;
            }
        });

        // Close modal when clicking X
        modalClose.addEventListener('click', () => {
            toolsModal.style.display = 'none';
        });

        // Close modal when clicking outside
        window.addEventListener('click', (e) => {
            if (e.target === toolsModal) {
                toolsModal.style.display = 'none';
            }
        });

        // Load list of available prompts on page load
        async function loadPromptsList() {
            try {
                const url = urlPrefix + '/nl-to-cql/list-prompts';
                const response = await fetch(url);
                if (response.ok) {
                    const data = await response.json();
                    promptSelect.innerHTML = '<option value="">-- Select a prompt file --</option>';
                    data.files.forEach(file => {
                        const option = document.createElement('option');
                        option.value = file.name;
                        option.textContent = file.name;
                        promptSelect.appendChild(option);
                    });
                }
            } catch (error) {
                console.error('Error loading prompts list:', error);
            }
        }

        // Load prompts list on page load
        loadPromptsList();

        // Handle form submission
        form.addEventListener('submit', async (e) => {
            e.preventDefault();

            const userInput = document.getElementById('userInput').value;
            const systemPrompt = systemPromptTextarea.value;

            if (!userInput) {
                alert('Please enter a natural language query');
                return;
            }

            // Show loading state
            submitBtn.disabled = true;
            submitBtn.innerHTML = 'Translating<span class="loading"></span>';
            resultBox.classList.remove('show');

            try {
                const url = urlPrefix + '/nl-to-cql';
                const response = await fetch(url, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({
                        userInput: userInput,
                        systemPrompt: systemPrompt
                    })
                });
                const data = await response.json();

                // Display results
                resultBox.classList.add('show');
                resultBox.classList.remove('error', 'success');

                if (response.ok) {
                    resultBox.classList.add('success');
                    resultContent.textContent = data.response;
                } else {
                    resultBox.classList.add('error');
                    resultContent.textContent = JSON.stringify(data, null, 2);
                }
            } catch (error) {
                resultBox.classList.add('show', 'error');
                resultContent.textContent = 'Error: ' + error.message;
            } finally {
                submitBtn.disabled = false;
                submitBtn.innerHTML = 'Translate to CQL';
            }
        });

        // Handle save system prompt
        savePromptBtn.addEventListener('click', async () => {
            const systemPrompt = systemPromptTextarea.value;
            const customName = promptName.value.trim();

            if (!systemPrompt.trim()) {
                alert('System prompt cannot be empty');
                return;
            }

            savePromptBtn.disabled = true;
            savePromptBtn.innerHTML = 'Saving<span class="loading"></span>';
            saveStatus.classList.remove('show');

            try {
                const url = urlPrefix + '/nl-to-cql/save-prompt';
                const response = await fetch(url, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({
                        content: systemPrompt,
                        name: customName
                    })
                });

                if (response.ok) {
                    const data = await response.json();
                    saveStatus.className = 'save-status show success-status';
                    saveStatus.textContent = '✓ System prompt saved: ' + data.filename;
                    // Clear the custom name input
                    promptName.value = '';
                    // Reload the prompts list
                    await loadPromptsList();
                } else {
                    const data = await response.json();
                    saveStatus.className = 'save-status show error-status';
                    saveStatus.textContent = '✗ Error: ' + (data.error || 'Failed to save prompt');
                }
            } catch (error) {
                saveStatus.className = 'save-status show error-status';
                saveStatus.textContent = '✗ Error: ' + error.message;
            } finally {
                savePromptBtn.disabled = false;
                savePromptBtn.innerHTML = 'Save System Prompt';
            }
        });

        // Handle prompt selection from dropdown
        promptSelect.addEventListener('change', async () => {
            const selectedFile = promptSelect.value;

            if (!selectedFile) {
                return;
            }

            saveStatus.classList.remove('show');

            try {
                const url = urlPrefix + '/nl-to-cql/load-prompt?file=' + encodeURIComponent(selectedFile);
                const response = await fetch(url);

                if (response.ok) {
                    const data = await response.json();
                    systemPromptTextarea.value = data.systemPrompt;
                    saveStatus.className = 'save-status show success-status';
                    saveStatus.textContent = '✓ System prompt loaded from: ' + data.source;
                } else {
                    const data = await response.json();
                    saveStatus.className = 'save-status show error-status';
                    saveStatus.textContent = '✗ Error: ' + (data.error || 'Failed to load prompt');
                    promptSelect.value = '';
                }
            } catch (error) {
                saveStatus.className = 'save-status show error-status';
                saveStatus.textContent = '✗ Error: ' + error.message;
                promptSelect.value = '';
            }
        });

        // Handle load default prompt button
        loadDefaultBtn.addEventListener('click', async () => {
            loadDefaultBtn.disabled = true;
            loadDefaultBtn.innerHTML = 'Loading<span class="loading"></span>';
            saveStatus.classList.remove('show');

            try {
                const url = urlPrefix + '/nl-to-cql/load-default-prompt';
                const response = await fetch(url);

                if (response.ok) {
                    const data = await response.json();
                    systemPromptTextarea.value = data.systemPrompt;
                    saveStatus.className = 'save-status show success-status';
                    saveStatus.textContent = '✓ System prompt loaded from: ' + data.source;
                    // Reset the select box
                    promptSelect.value = '';
                } else {
                    const data = await response.json();
                    saveStatus.className = 'save-status show error-status';
                    saveStatus.textContent = '✗ Error: ' + (data.error || 'Failed to load default prompt');
                }
            } catch (error) {
                saveStatus.className = 'save-status show error-status';
                saveStatus.textContent = '✗ Error: ' + error.message;
            } finally {
                loadDefaultBtn.disabled = false;
                loadDefaultBtn.innerHTML = 'Load Default (sysprompt.txt)';
            }
        });
    </script>
</body>
</html>`,
		modelName,
		initialSystemPrompt,
		api.version.Version,
		api.version.BuildDate,
		urlPrefix)

	ctx.Header("Content-Type", "text/html; charset=utf-8")
	ctx.String(http.StatusOK, html)
}
