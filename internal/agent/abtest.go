// Package agent provides agent orchestration capabilities
package agent

import (
	"context"
	"fmt"
	"time"
)

// ABTest runs parallel agents and compares results
type ABTest struct {
	agentA       *Agent
	agentB       *Agent
	comparator   *Comparator
}

// ABConfig configures the A/B test
type ABConfig struct {
	Task        string
	ModelA      string
	ModelB      string
	Timeout     time.Duration
	Criteria    []string // "cost", "time", "quality"
}

// ABResult represents the results of an A/B test
type ABResult struct {
	Winner      string // "A", "B", or "tie"
	AgentA      AgentResult
	AgentB      AgentResult
	Comparison  ComparisonResult
	Duration    time.Duration
}

// AgentResult represents the result from a single agent
type AgentResult struct {
	AgentID      string
	Model        string
	Output       string
	Cost         float64
	Duration     time.Duration
	TokenUsage   int
	QualityScore float64
}

// ComparisonResult represents the comparison between two agents
type ComparisonResult struct {
	CostWinner      string
	TimeWinner      string
	QualityWinner   string
	OverallWinner   string
	DiffSummary     string
}

// NewABTest creates a new A/B test
func NewABTest() *ABTest {
	return &ABTest{
		comparator: NewComparator(),
	}
}

// Run executes the A/B test
func (ab *ABTest) Run(ctx context.Context, config ABConfig) (*ABResult, error) {
	startTime := time.Now()
	result := &ABResult{}

	fmt.Println("🧪 Starting A/B Comparison...")
	fmt.Printf("Model A: %s vs Model B: %s\n\n", config.ModelA, config.ModelB)

	// Create timeout context
	ctx, cancel := context.WithTimeout(ctx, config.Timeout)
	defer cancel()

	// Run agents in parallel
	resultChan := make(chan AgentResult, 2)

	go func() {
		res := ab.runAgent(ctx, "A", config.ModelA, config.Task)
		resultChan <- res
	}()

	go func() {
		res := ab.runAgent(ctx, "B", config.ModelB, config.Task)
		resultChan <- res
	}()

	// Collect results
	resA := <-resultChan
	resB := <-resultChan

	result.AgentA = resA
	result.AgentB = resB
	result.Duration = time.Since(startTime)

	// Compare results
	result.Comparison = ab.compareResults(resA, resB)
	result.Winner = result.Comparison.OverallWinner

	// Print results
	ab.printResults(result)

	return result, nil
}

// runAgent runs a single agent
func (ab *ABTest) runAgent(ctx context.Context, agentID, model, task string) AgentResult {
	startTime := time.Now()

	// In real implementation, this would call the LLM
	// Simulate work
	select {
	case <-ctx.Done():
		return AgentResult{
			AgentID:  agentID,
			Model:    model,
			Duration: time.Since(startTime),
		}
	case <-time.After(100 * time.Millisecond):
		// Continue
	}

	return AgentResult{
		AgentID:      agentID,
		Model:        model,
		Output:       fmt.Sprintf("Output from %s using %s", agentID, model),
		Cost:         0.50 + float64(len(task))*0.01,
		Duration:     time.Since(startTime),
		TokenUsage:   1000 + len(task)*10,
		QualityScore: 0.85,
	}
}

// compareResults compares two agent results
func (ab *ABTest) compareResults(a, b AgentResult) ComparisonResult {
	comp := ComparisonResult{}

	// Cost comparison
	if a.Cost < b.Cost {
		comp.CostWinner = "A"
	} else if b.Cost < a.Cost {
		comp.CostWinner = "B"
	} else {
		comp.CostWinner = "tie"
	}

	// Time comparison
	if a.Duration < b.Duration {
		comp.TimeWinner = "A"
	} else if b.Duration < a.Duration {
		comp.TimeWinner = "B"
	} else {
		comp.TimeWinner = "tie"
	}

	// Quality comparison
	if a.QualityScore > b.QualityScore {
		comp.QualityWinner = "A"
	} else if b.QualityScore > a.QualityScore {
		comp.QualityWinner = "B"
	} else {
		comp.QualityWinner = "tie"
	}

	// Overall winner (weighted)
	scoreA := 0
	scoreB := 0

	if comp.CostWinner == "A" {
		scoreA++
	} else if comp.CostWinner == "B" {
		scoreB++
	}

	if comp.TimeWinner == "A" {
		scoreA++
	} else if comp.TimeWinner == "B" {
		scoreB++
	}

	if comp.QualityWinner == "A" {
		scoreA += 2 // Quality weighted more
	} else if comp.QualityWinner == "B" {
		scoreB += 2
	}

	if scoreA > scoreB {
		comp.OverallWinner = "A"
	} else if scoreB > scoreA {
		comp.OverallWinner = "B"
	} else {
		comp.OverallWinner = "tie"
	}

	comp.DiffSummary = fmt.Sprintf("Cost: %s wins | Time: %s wins | Quality: %s wins",
		comp.CostWinner, comp.TimeWinner, comp.QualityWinner)

	return comp
}

// printResults prints the comparison results
func (ab *ABTest) printResults(result *ABResult) {
	fmt.Println("📊 A/B Test Results")
	fmt.Println("==================")
	fmt.Printf("\nAgent A (%s):\n", result.AgentA.Model)
	fmt.Printf("  Cost: $%.2f\n", result.AgentA.Cost)
	fmt.Printf("  Time: %v\n", result.AgentA.Duration)
	fmt.Printf("  Quality: %.1f%%\n", result.AgentA.QualityScore*100)
	fmt.Printf("  Tokens: %d\n", result.AgentA.TokenUsage)

	fmt.Printf("\nAgent B (%s):\n", result.AgentB.Model)
	fmt.Printf("  Cost: $%.2f\n", result.AgentB.Cost)
	fmt.Printf("  Time: %v\n", result.AgentB.Duration)
	fmt.Printf("  Quality: %.1f%%\n", result.AgentB.QualityScore*100)
	fmt.Printf("  Tokens: %d\n", result.AgentB.TokenUsage)

	fmt.Println("\nComparison:")
	fmt.Printf("  %s\n", result.Comparison.DiffSummary)
	fmt.Printf("\n🏆 Winner: Agent %s\n", result.Winner)
}

// Comparator compares agent outputs
type Comparator struct{}

// NewComparator creates a new comparator
func NewComparator() *Comparator {
	return &Comparator{}
}

// CompareOutputs compares two outputs and returns a similarity score
func (c *Comparator) CompareOutputs(outputA, outputB string) float64 {
	// Simple similarity calculation
	// In real implementation, this would use more sophisticated methods
	if outputA == outputB {
		return 1.0
	}

	// Calculate word overlap
	wordsA := tokenize(outputA)
	wordsB := tokenize(outputB)

	if len(wordsA) == 0 || len(wordsB) == 0 {
		return 0.0
	}

	overlap := 0
	wordSet := make(map[string]bool)
	for _, w := range wordsA {
		wordSet[w] = true
	}

	for _, w := range wordsB {
		if wordSet[w] {
			overlap++
		}
	}

	return float64(overlap*2) / float64(len(wordsA)+len(wordsB))
}

func tokenize(text string) []string {
	// Simple tokenization
	var tokens []string
	var current []rune

	for _, r := range text {
		if r == ' ' || r == '\n' || r == '\t' {
			if len(current) > 0 {
				tokens = append(tokens, string(current))
				current = nil
			}
		} else {
			current = append(current, r)
		}
	}

	if len(current) > 0 {
		tokens = append(tokens, string(current))
	}

	return tokens
}
