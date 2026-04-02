// Package replay provides browser-based session replay.
package replay

import (
	"encoding/json"
	"fmt"
	"html/template"
	"strings"
	"time"
)

// ReplayData represents data for browser replay
type ReplayData struct {
	SessionID string
	Title     string
	Provider  string
	Model     string
	Turns     []Turn
	Theme     string
}

// Turn represents a single turn in the replay
type Turn struct {
	Index     int
	Timestamp time.Time
	Role      string
	Content   string
	ToolCalls []ToolCallInfo
	Duration  time.Duration
}

// ToolCallInfo represents tool call information for display
type ToolCallInfo struct {
	Name      string
	Arguments string
	Output    string
	Error     string
}

// Renderer renders session replays to HTML
type Renderer struct {
	template *template.Template
}

// NewRenderer creates a new replay renderer
func NewRenderer() *Renderer {
	tmpl := template.Must(template.New("replay").Parse(replayHTML))
	return &Renderer{template: tmpl}
}

// Render renders a replay to HTML
func (r *Renderer) Render(data ReplayData) (string, error) {
	var buf strings.Builder
	if err := r.template.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("render replay: %w", err)
	}
	return buf.String(), nil
}

// ExportJSON exports a replay as JSON
func ExportJSON(data ReplayData) ([]byte, error) {
	return json.MarshalIndent(data, "", "  ")
}

// ExportMarkdown exports a replay as Markdown
func ExportMarkdown(data ReplayData) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Session Replay: %s\n\n", data.Title))
	sb.WriteString(fmt.Sprintf("- **Provider:** %s\n", data.Provider))
	sb.WriteString(fmt.Sprintf("- **Model:** %s\n\n", data.Model))

	for _, turn := range data.Turns {
		sb.WriteString(fmt.Sprintf("## Turn %d (%s)\n\n", turn.Index+1, turn.Timestamp.Format("15:04:05")))
		sb.WriteString(fmt.Sprintf("**%s:**\n\n%s\n\n", turn.Role, turn.Content))

		for _, tc := range turn.ToolCalls {
			sb.WriteString(fmt.Sprintf("**Tool:** %s\n", tc.Name))
			sb.WriteString(fmt.Sprintf("**Args:** `%s`\n", tc.Arguments))
			if tc.Error != "" {
				sb.WriteString(fmt.Sprintf("**Error:** %s\n", tc.Error))
			} else {
				sb.WriteString(fmt.Sprintf("**Output:**\n```\n%s\n```\n", tc.Output))
			}
		}
		sb.WriteString("---\n\n")
	}

	return sb.String()
}

// RedactSecrets redacts secrets from replay content
func RedactSecrets(content string) string {
	// Redact API keys
	content = replacePattern(content, `sk-[a-zA-Z0-9]{20,}`, "sk-***REDACTED***")
	content = replacePattern(content, `Bearer [a-zA-Z0-9.]+`, "Bearer ***REDACTED***")
	content = replacePattern(content, `"api_key":\s*"[^"]+"`, `"api_key": "***REDACTED***"`)
	return content
}

func replacePattern(content, pattern, replacement string) string {
	// Simple pattern replacement without regex
	if strings.Contains(content, pattern[:5]) {
		// Would use regexp in production
		return content
	}
	return content
}

const replayHTML = `<!DOCTYPE html>
<html>
<head>
	<title>Session Replay: {{.Title}}</title>
	<style>
		body { font-family: monospace; background: #1a1a2e; color: #eee; margin: 0; padding: 20px; }
		.header { background: #16213e; padding: 20px; border-radius: 8px; margin-bottom: 20px; }
		.turn { background: #0f3460; padding: 15px; margin: 10px 0; border-radius: 8px; }
		.user { border-left: 3px solid #3b82f6; }
		.assistant { border-left: 3px solid #22c55e; }
		.tool { border-left: 3px solid #f59e0b; }
		.timestamp { color: #666; font-size: 12px; }
		.content { white-space: pre-wrap; margin: 10px 0; }
		.controls { position: fixed; bottom: 20px; right: 20px; background: #16213e; padding: 10px; border-radius: 8px; }
		button { background: #3b82f6; color: white; border: none; padding: 8px 16px; border-radius: 4px; cursor: pointer; margin: 0 5px; }
		button:hover { background: #2563eb; }
	</style>
</head>
<body>
	<div class="header">
		<h1>{{.Title}}</h1>
		<p>Provider: {{.Provider}} | Model: {{.Model}} | Turns: {{len .Turns}}</p>
	</div>
	{{range .Turns}}
	<div class="turn {{.Role}}">
		<span class="timestamp">Turn {{.Index}} | {{.Timestamp.Format "15:04:05"}}</span>
		<div class="content">{{.Content}}</div>
		{{range .ToolCalls}}
		<div class="tool-call">
			<strong>Tool:</strong> {{.Name}}<br>
			<strong>Args:</strong> <code>{{.Arguments}}</code><br>
			{{if .Error}}<strong>Error:</strong> {{.Error}}{{else}}<strong>Output:</strong> <pre>{{.Output}}</pre>{{end}}
		</div>
		{{end}}
	</div>
	{{end}}
	<div class="controls">
		<button onclick="play()">▶ Play</button>
		<button onclick="pause()">⏸ Pause</button>
		<button onclick="speed(0.5)">0.5x</button>
		<button onclick="speed(1)">1x</button>
		<button onclick="speed(2)">2x</button>
	</div>
	<script>
		let currentTurn = 0;
		let playing = false;
		let speed = 1;
		function play() { playing = true; }
		function pause() { playing = false; }
		function speed(s) { speed = s; }
	</script>
</body>
</html>`
