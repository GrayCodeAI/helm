// Package web provides the web dashboard server
package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/yourname/helm/internal/db"
)

// Server provides the web dashboard
type Server struct {
	querier    db.Querier
	mux        *http.ServeMux
	addr       string
	sseClients []chan string
	sseMu      sync.RWMutex
	startedAt  time.Time
}

// NewServer creates a new web server
func NewServer(querier db.Querier, addr string) *Server {
	s := &Server{
		querier:   querier,
		mux:       http.NewServeMux(),
		addr:      addr,
		startedAt: time.Now(),
	}

	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	s.mux.HandleFunc("/", s.handleIndex)
	s.mux.HandleFunc("/sessions", s.handleSessions)
	s.mux.HandleFunc("/sessions/", s.handleSessionDetail)
	s.mux.HandleFunc("/cost", s.handleCost)
	s.mux.HandleFunc("/memory", s.handleMemory)
	s.mux.HandleFunc("/analytics", s.handleAnalytics)
	s.mux.HandleFunc("/api/sessions", s.handleAPISessions)
	s.mux.HandleFunc("/api/cost", s.handleAPICost)
	s.mux.HandleFunc("/api/memory", s.handleAPIMemory)
	s.mux.HandleFunc("/api/analytics", s.handleAPIAnalytics)
	s.mux.HandleFunc("/sse/events", s.handleSSE)
	s.mux.Handle("/static/", http.FileServer(http.Dir("static")))
}

// Start starts the web server
func (s *Server) Start() error {
	return http.ListenAndServe(s.addr, s)
}

// StartWithContext starts the web server with context
func (s *Server) StartWithContext(ctx context.Context) error {
	server := &http.Server{
		Addr:    s.addr,
		Handler: s,
	}

	go func() {
		<-ctx.Done()
		server.Shutdown(context.Background())
	}()

	return server.ListenAndServe()
}

// ServeHTTP implements http.Handler
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// broadcastSSE sends a message to all SSE clients
func (s *Server) _broadcastSSE(msg string) {
	s.sseMu.RLock()
	defer s.sseMu.RUnlock()
	for _, ch := range s.sseClients {
		select {
		case ch <- msg:
		default:
		}
	}
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	sessions, _ := s.querier.ListRecentSessions(r.Context(), 10)
	uptime := time.Since(s.startedAt).Round(time.Second)

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
	<title>HELM Dashboard</title>
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<style>
		* { box-sizing: border-box; margin: 0; padding: 0; }
		body { font-family: -apple-system, BlinkMacSystemFont, sans-serif; background: #f5f5f5; }
		nav { background: #1a1a2e; padding: 15px 20px; display: flex; gap: 20px; align-items: center; }
		nav a { color: #eee; text-decoration: none; padding: 8px 16px; border-radius: 4px; }
		nav a:hover { background: #16213e; }
		nav a.active { background: #0f3460; color: #fff; }
		.container { max-width: 1200px; margin: 20px auto; padding: 0 20px; }
		.stats { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 15px; margin-bottom: 20px; }
		.stat-card { background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
		.stat-card h3 { color: #666; font-size: 14px; margin-bottom: 8px; }
		.stat-card .value { font-size: 28px; font-weight: bold; color: #1a1a2e; }
		.section { background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); margin-bottom: 20px; }
		.section h2 { margin-bottom: 15px; color: #1a1a2e; }
		table { width: 100%%; border-collapse: collapse; }
		th, td { padding: 12px; text-align: left; border-bottom: 1px solid #eee; }
		th { background: #f8f8f8; font-weight: 600; }
		.status-running { color: #22c55e; }
		.status-done { color: #3b82f6; }
		.status-failed { color: #ef4444; }
		.status-paused { color: #f59e0b; }
	</style>
</head>
<body>
	<nav>
		<strong style="color:#fff;margin-right:20px;">HELM</strong>
		<a href="/" class="active">Dashboard</a>
		<a href="/sessions">Sessions</a>
		<a href="/cost">Cost</a>
		<a href="/memory">Memory</a>
		<a href="/analytics">Analytics</a>
	</nav>
	<div class="container">
		<div class="stats">
			<div class="stat-card">
				<h3>Total Sessions</h3>
				<div class="value">%d</div>
			</div>
			<div class="stat-card">
				<h3>Uptime</h3>
				<div class="value">%s</div>
			</div>
			<div class="stat-card">
				<h3>Running</h3>
				<div class="value">%d</div>
			</div>
			<div class="stat-card">
				<h3>Completed</h3>
				<div class="value">%d</div>
			</div>
		</div>
		<div class="section">
			<h2>Recent Sessions</h2>
			<table>
				<tr><th>ID</th><th>Status</th><th>Provider</th><th>Model</th><th>Cost</th><th>Started</th></tr>`,
		len(sessions), uptime,
		countByStatus(sessions, "running"),
		countByStatus(sessions, "done"))

	for _, sess := range sessions {
		fmt.Fprintf(w, `<tr>
			<td><a href="/sessions/%s">%s</a></td>
			<td class="status-%s">%s</td>
			<td>%s</td>
			<td>%s</td>
			<td>$%.4f</td>
			<td>%s</td>
		</tr>`,
			sess.ID[:8], sess.ID[:12],
			sess.Status, sess.Status,
			sess.Provider, sess.Model,
			sess.Cost, sess.StartedAt[:19])
	}

	fmt.Fprintf(w, `</table>
		</div>
	</div>
</body>
</html>`)
}

func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	sessions, _ := s.querier.ListRecentSessions(r.Context(), 100)

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
	<title>Sessions - HELM</title>
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<style>
		* { box-sizing: border-box; margin: 0; padding: 0; }
		body { font-family: -apple-system, BlinkMacSystemFont, sans-serif; background: #f5f5f5; }
		nav { background: #1a1a2e; padding: 15px 20px; display: flex; gap: 20px; align-items: center; }
		nav a { color: #eee; text-decoration: none; padding: 8px 16px; border-radius: 4px; }
		nav a:hover { background: #16213e; }
		nav a.active { background: #0f3460; color: #fff; }
		.container { max-width: 1200px; margin: 20px auto; padding: 0 20px; }
		.section { background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
		table { width: 100%%; border-collapse: collapse; }
		th, td { padding: 12px; text-align: left; border-bottom: 1px solid #eee; }
		th { background: #f8f8f8; }
		.status-running { color: #22c55e; }
		.status-done { color: #3b82f6; }
		.status-failed { color: #ef4444; }
		.status-paused { color: #f59e0b; }
	</style>
</head>
<body>
	<nav>
		<strong style="color:#fff;margin-right:20px;">HELM</strong>
		<a href="/">Dashboard</a>
		<a href="/sessions" class="active">Sessions</a>
		<a href="/cost">Cost</a>
		<a href="/memory">Memory</a>
		<a href="/analytics">Analytics</a>
	</nav>
	<div class="container">
		<div class="section">
			<h2>All Sessions (%d)</h2>
			<table>
				<tr><th>ID</th><th>Status</th><th>Provider</th><th>Model</th><th>Cost</th><th>Tokens</th><th>Started</th></tr>`,
		len(sessions))

	for _, sess := range sessions {
		totalTokens := sess.InputTokens + sess.OutputTokens
		fmt.Fprintf(w, `<tr>
			<td><a href="/sessions/%s">%s</a></td>
			<td class="status-%s">%s</td>
			<td>%s</td>
			<td>%s</td>
			<td>$%.4f</td>
			<td>%d</td>
			<td>%s</td>
		</tr>`,
			sess.ID[:8], sess.ID[:12],
			sess.Status, sess.Status,
			sess.Provider, sess.Model,
			sess.Cost, totalTokens, sess.StartedAt[:19])
	}

	fmt.Fprintf(w, `</table></div></div></body></html>`)
}

func (s *Server) handleSessionDetail(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/sessions/"):]
	sess, err := s.querier.GetSession(r.Context(), id)
	if err != nil {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	messages, _ := s.querier.GetMessagesBySession(r.Context(), sess.ID)
	changes, _ := s.querier.ListFileChanges(r.Context(), sess.ID)

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
	<title>Session %s - HELM</title>
	<style>
		body { font-family: monospace; padding: 20px; background: #1a1a2e; color: #eee; }
		.container { max-width: 900px; margin: 0 auto; }
		.info { background: #16213e; padding: 15px; border-radius: 8px; margin-bottom: 20px; }
		.msg { background: #0f3460; padding: 10px; margin: 10px 0; border-radius: 4px; }
		.msg-user { border-left: 3px solid #3b82f6; }
		.msg-assistant { border-left: 3px solid #22c55e; }
		.file { background: #16213e; padding: 8px; margin: 5px 0; border-radius: 4px; }
	</style>
</head>
<body>
	<div class="container">
		<h1>Session %s</h1>
		<div class="info">
			<p><b>Status:</b> %s | <b>Provider:</b> %s | <b>Model:</b> %s</p>
			<p><b>Cost:</b> $%.4f | <b>Tokens:</b> %d input / %d output</p>
			<p><b>Started:</b> %s</p>
		</div>
		<h2>Messages (%d)</h2>`,
		id[:8], id[:8],
		sess.Status, sess.Provider, sess.Model,
		sess.Cost, sess.InputTokens, sess.OutputTokens, sess.StartedAt,
		len(messages))

	for _, msg := range messages {
		cls := "msg-user"
		if msg.Role == "assistant" {
			cls = "msg-assistant"
		}
		content := msg.Content
		if len(content) > 200 {
			content = content[:200] + "..."
		}
		fmt.Fprintf(w, `<div class="msg %s"><b>%s:</b> %s</div>`, cls, msg.Role, content)
	}

	fmt.Fprintf(w, `<h2>File Changes (%d)</h2>`, len(changes))
	for _, c := range changes {
		fmt.Fprintf(w, `<div class="file">%s (+%d/-%d) %s</div>`,
			c.FilePath, c.Additions.Int64, c.Deletions.Int64, c.Classification.String)
	}

	fmt.Fprintf(w, `</div></body></html>`)
}

func (s *Server) handleCost(w http.ResponseWriter, r *http.Request) {
	project := r.URL.Query().Get("project")
	if project == "" {
		project = "default"
	}

	today, _ := s.querier.GetCostByProjectToday(r.Context(), project)
	week, _ := s.querier.GetCostByProjectWeek(r.Context(), project)
	month, _ := s.querier.GetCostByProjectMonth(r.Context(), project)
	budget, _ := s.querier.GetBudget(r.Context(), project)
	records, _ := s.querier.ListCostRecords(r.Context(), db.ListCostRecordsParams{Project: project, Limit: 50})

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
	<title>Cost - HELM</title>
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<style>
		* { box-sizing: border-box; margin: 0; padding: 0; }
		body { font-family: -apple-system, BlinkMacSystemFont, sans-serif; background: #f5f5f5; }
		nav { background: #1a1a2e; padding: 15px 20px; display: flex; gap: 20px; align-items: center; }
		nav a { color: #eee; text-decoration: none; padding: 8px 16px; border-radius: 4px; }
		nav a:hover { background: #16213e; }
		nav a.active { background: #0f3460; color: #fff; }
		.container { max-width: 1200px; margin: 20px auto; padding: 0 20px; }
		.stats { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 15px; margin-bottom: 20px; }
		.stat-card { background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
		.stat-card h3 { color: #666; font-size: 14px; margin-bottom: 8px; }
		.stat-card .value { font-size: 28px; font-weight: bold; color: #1a1a2e; }
		.section { background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); margin-bottom: 20px; }
		table { width: 100%%; border-collapse: collapse; }
		th, td { padding: 12px; text-align: left; border-bottom: 1px solid #eee; }
		th { background: #f8f8f8; }
		.progress { background: #eee; height: 20px; border-radius: 10px; overflow: hidden; margin: 10px 0; }
		.progress-bar { background: #3b82f6; height: 100%%; transition: width 0.3s; }
	</style>
</head>
<body>
	<nav>
		<strong style="color:#fff;margin-right:20px;">HELM</strong>
		<a href="/">Dashboard</a>
		<a href="/sessions">Sessions</a>
		<a href="/cost" class="active">Cost</a>
		<a href="/memory">Memory</a>
		<a href="/analytics">Analytics</a>
	</nav>
	<div class="container">
		<div class="stats">
			<div class="stat-card"><h3>Today</h3><div class="value">$%.4f</div></div>
			<div class="stat-card"><h3>This Week</h3><div class="value">$%.4f</div></div>
			<div class="stat-card"><h3>This Month</h3><div class="value">$%.4f</div></div>
			<div class="stat-card"><h3>Daily Budget</h3><div class="value">$%.2f</div></div>
		</div>
		<div class="section">
			<h2>Budget Usage</h2>
			<div class="progress"><div class="progress-bar" style="width: %.0f%%"></div></div>
			<p>$%.2f of $%.2f daily budget used</p>
		</div>
		<div class="section">
			<h2>Cost Records</h2>
			<table>
				<tr><th>Session</th><th>Provider</th><th>Model</th><th>Input</th><th>Output</th><th>Cost</th></tr>`,
		today.TotalCost, week.TotalCost, month.TotalCost,
		budget.DailyLimit.Float64,
		budgetPercent(today.TotalCost, budget.DailyLimit.Float64),
		today.TotalCost, budget.DailyLimit.Float64)

	for _, rec := range records {
		fmt.Fprintf(w, `<tr>
			<td>%s</td><td>%s</td><td>%s</td>
			<td>%d</td><td>%d</td><td>$%.4f</td>
		</tr>`,
			rec.SessionID[:8], rec.Provider, rec.Model,
			rec.InputTokens.Int64, rec.OutputTokens.Int64, rec.TotalCost.Float64)
	}

	fmt.Fprintf(w, `</table></div></div></body></html>`)
}

func (s *Server) handleMemory(w http.ResponseWriter, r *http.Request) {
	project := r.URL.Query().Get("project")
	if project == "" {
		project = "default"
	}

	memories, _ := s.querier.ListMemories(r.Context(), project)
	types := map[string]int{}
	for _, m := range memories {
		types[m.Type]++
	}

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, `<!DOCTYPE html>
<html>
<head>
	<title>Memory - HELM</title>
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<style>
		* { box-sizing: border-box; margin: 0; padding: 0; }
		body { font-family: -apple-system, BlinkMacSystemFont, sans-serif; background: #f5f5f5; }
		nav { background: #1a1a2e; padding: 15px 20px; display: flex; gap: 20px; align-items: center; }
		nav a { color: #eee; text-decoration: none; padding: 8px 16px; border-radius: 4px; }
		nav a:hover { background: #16213e; }
		nav a.active { background: #0f3460; color: #fff; }
		.container { max-width: 1200px; margin: 20px auto; padding: 0 20px; }
		.stats { display: grid; grid-template-columns: repeat(auto-fit, minmax(150px, 1fr)); gap: 15px; margin-bottom: 20px; }
		.stat-card { background: white; padding: 15px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); text-align: center; }
		.stat-card .value { font-size: 24px; font-weight: bold; color: #1a1a2e; }
		.stat-card h3 { color: #666; font-size: 12px; }
		.section { background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
		table { width: 100%%; border-collapse: collapse; }
		th, td { padding: 12px; text-align: left; border-bottom: 1px solid #eee; }
		th { background: #f8f8f8; }
		.confidence { display: inline-block; padding: 2px 8px; border-radius: 12px; font-size: 12px; }
		.conf-high { background: #dcfce7; color: #166534; }
		.conf-med { background: #fef3c7; color: #92400e; }
		.conf-low { background: #fee2e2; color: #991b1b; }
	</style>
</head>
<body>
	<nav>
		<strong style="color:#fff;margin-right:20px;">HELM</strong>
		<a href="/">Dashboard</a>
		<a href="/sessions">Sessions</a>
		<a href="/cost">Cost</a>
		<a href="/memory" class="active">Memory</a>
		<a href="/analytics">Analytics</a>
	</nav>
	<div class="container">
		<div class="stats">`)
	fmt.Fprintf(w, `%d`, len(memories))

	for t, count := range types {
		fmt.Fprintf(w, `<div class="stat-card"><div class="value">%d</div><h3>%s</h3></div>`, count, t)
	}

	fmt.Fprintf(w, `</div>
		<div class="section">
			<h2>Memory Entries (%d)</h2>
			<table>
				<tr><th>Type</th><th>Key</th><th>Value</th><th>Confidence</th><th>Usage</th><th>Source</th></tr>`,
		len(memories))

	for _, m := range memories {
		cls := "conf-med"
		if m.Confidence > 0.7 {
			cls = "conf-high"
		} else if m.Confidence < 0.4 {
			cls = "conf-low"
		}
		fmt.Fprintf(w, `<tr>
			<td>%s</td><td>%s</td><td>%s</td>
			<td><span class="confidence %s">%.0f%%</span></td>
			<td>%d</td><td>%s</td>
		</tr>`,
			m.Type, m.Key, truncate(m.Value, 50),
			cls, m.Confidence*100, m.UsageCount, m.Source)
	}

	fmt.Fprintf(w, `</table></div></div></body></html>`)
}

func (s *Server) handleAnalytics(w http.ResponseWriter, r *http.Request) {
	performances, _ := s.querier.ListModelPerformance(r.Context())

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, `<!DOCTYPE html>
<html>
<head>
	<title>Analytics - HELM</title>
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<style>
		* { box-sizing: border-box; margin: 0; padding: 0; }
		body { font-family: -apple-system, BlinkMacSystemFont, sans-serif; background: #f5f5f5; }
		nav { background: #1a1a2e; padding: 15px 20px; display: flex; gap: 20px; align-items: center; }
		nav a { color: #eee; text-decoration: none; padding: 8px 16px; border-radius: 4px; }
		nav a:hover { background: #16213e; }
		nav a.active { background: #0f3460; color: #fff; }
		.container { max-width: 1200px; margin: 20px auto; padding: 0 20px; }
		.section { background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); margin-bottom: 20px; }
		table { width: 100%%; border-collapse: collapse; }
		th, td { padding: 12px; text-align: left; border-bottom: 1px solid #eee; }
		th { background: #f8f8f8; }
		.bar { background: #3b82f6; height: 20px; border-radius: 4px; }
	</style>
</head>
<body>
	<nav>
		<strong style="color:#fff;margin-right:20px;">HELM</strong>
		<a href="/">Dashboard</a>
		<a href="/sessions">Sessions</a>
		<a href="/cost">Cost</a>
		<a href="/memory">Memory</a>
		<a href="/analytics" class="active">Analytics</a>
	</nav>
	<div class="container">
		<div class="section">
			<h2>Model Performance</h2>
			<table>
				<tr><th>Model</th><th>Task Type</th><th>Attempts</th><th>Successes</th><th>Success Rate</th><th>Avg Cost</th><th>Avg Tokens</th></tr>`,
		len(performances))

	for _, p := range performances {
		rate := 0.0
		if p.Attempts > 0 {
			rate = float64(p.Successes) / float64(p.Attempts) * 100
		}
		fmt.Fprintf(w, `<tr>
			<td>%s</td><td>%s</td><td>%d</td><td>%d</td>
			<td><div style="background:#eee;width:100px;display:inline-block"><div class="bar" style="width:%.0f%%"></div></div> %.0f%%</td>
			<td>$%.4f</td><td>%d</td>
		</tr>`,
			p.Model, p.TaskType, p.Attempts, p.Successes,
			rate, rate, p.TotalCost/float64(p.Attempts), p.AvgTokens)
	}

	fmt.Fprintf(w, `</table></div></div></body></html>`)
}

// API Handlers

func (s *Server) handleAPISessions(w http.ResponseWriter, r *http.Request) {
	sessions, err := s.querier.ListRecentSessions(r.Context(), 100)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sessions)
}

func (s *Server) handleAPICost(w http.ResponseWriter, r *http.Request) {
	project := r.URL.Query().Get("project")
	if project == "" {
		project = "default"
	}

	today, _ := s.querier.GetCostByProjectToday(r.Context(), project)
	week, _ := s.querier.GetCostByProjectWeek(r.Context(), project)
	month, _ := s.querier.GetCostByProjectMonth(r.Context(), project)
	budget, _ := s.querier.GetBudget(r.Context(), project)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"today":  today,
		"week":   week,
		"month":  month,
		"budget": budget,
	})
}

func (s *Server) handleAPIMemory(w http.ResponseWriter, r *http.Request) {
	project := r.URL.Query().Get("project")
	if project == "" {
		project = "default"
	}

	memories, err := s.querier.ListMemories(r.Context(), project)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(memories)
}

func (s *Server) handleAPIAnalytics(w http.ResponseWriter, r *http.Request) {
	performances, err := s.querier.ListModelPerformance(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(performances)
}

func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	ch := make(chan string, 10)
	s.sseMu.Lock()
	s.sseClients = append(s.sseClients, ch)
	s.sseMu.Unlock()

	defer func() {
		s.sseMu.Lock()
		for i, c := range s.sseClients {
			if c == ch {
				s.sseClients = append(s.sseClients[:i], s.sseClients[i+1:]...)
				break
			}
		}
		s.sseMu.Unlock()
	}()

	// Send initial connection event
	fmt.Fprintf(w, "data: {\"type\":\"connected\",\"time\":\"%s\"}\n\n", time.Now().Format(time.RFC3339))
	flusher.Flush()

	// Keep connection open and forward events
	for {
		select {
		case msg := <-ch:
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

// Helper functions

func countByStatus(sessions []db.Session, status string) int {
	count := 0
	for _, s := range sessions {
		if s.Status == status {
			count++
		}
	}
	return count
}

func budgetPercent(spent, limit float64) float64 {
	if limit <= 0 {
		return 0
	}
	pct := (spent / limit) * 100
	if pct > 100 {
		return 100
	}
	return pct
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
