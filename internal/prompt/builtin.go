package prompt

// BuiltinPrompts returns all built-in prompt templates.
func BuiltinPrompts() []*PromptTemplate {
	return []*PromptTemplate{
		{
			Name:        "add-feature",
			Description: "Add a new feature to the codebase",
			Tags:        []string{"feature", "implementation"},
			Complexity:  "medium",
			Context:     []string{"project_memory", "file_structure"},
			Template: `Add the following feature: {{.Task}}

Guidelines:
- Follow existing conventions and patterns
- Write tests for new code
- Use existing abstractions where possible
- Keep changes focused and minimal`,
			Variables: []PromptVariable{
				{Name: "Task", Description: "Describe the feature to add", Required: true},
			},
			Source: "builtin",
		},
		{
			Name:        "fix-bug",
			Description: "Fix a specific bug in the codebase",
			Tags:        []string{"bug", "fix"},
			Complexity:  "low",
			Context:     []string{"project_memory"},
			Template: `Fix the following bug: {{.Bug}}

Context:
{{.Context}}

Guidelines:
- Understand the root cause before fixing
- Add a test case that reproduces the bug
- Keep the fix minimal and focused`,
			Variables: []PromptVariable{
				{Name: "Bug", Description: "Describe the bug to fix", Required: true},
				{Name: "Context", Description: "Additional context about the bug", Required: false},
			},
			Source: "builtin",
		},
		{
			Name:        "refactor",
			Description: "Refactor code with specific goals",
			Tags:        []string{"refactor", "cleanup"},
			Complexity:  "medium",
			Context:     []string{"project_memory"},
			Template: `Refactor the following code: {{.Target}}

Goals:
{{.Goals}}

Guidelines:
- Preserve existing behavior
- Improve readability and maintainability
- Follow existing patterns
- Update tests if needed`,
			Variables: []PromptVariable{
				{Name: "Target", Description: "The code or file to refactor", Required: true},
				{Name: "Goals", Description: "Refactoring goals (one per line)", Required: true},
			},
			Source: "builtin",
		},
		{
			Name:        "write-tests",
			Description: "Generate tests for existing code",
			Tags:        []string{"testing", "quality"},
			Complexity:  "low",
			Context:     []string{"project_memory"},
			Template: `Write tests for: {{.Target}}

Guidelines:
- Cover happy path and edge cases
- Follow existing test patterns
- Use table-driven tests where appropriate
- Aim for meaningful test names`,
			Variables: []PromptVariable{
				{Name: "Target", Description: "The code or file to test", Required: true},
			},
			Source: "builtin",
		},
		{
			Name:        "review",
			Description: "Review code for issues",
			Tags:        []string{"review", "quality"},
			Complexity:  "low",
			Context:     []string{},
			Template: `Review the following code for issues: {{.Target}}

Focus areas:
- Correctness and edge cases
- Security vulnerabilities
- Performance concerns
- Code readability
- Error handling

Provide specific, actionable feedback.`,
			Variables: []PromptVariable{
				{Name: "Target", Description: "The code or file to review", Required: true},
			},
			Source: "builtin",
		},
		{
			Name:        "explain",
			Description: "Explain complex code",
			Tags:        []string{"explanation", "learning"},
			Complexity:  "low",
			Context:     []string{},
			Template: `Explain the following code: {{.Target}}

Provide:
- What the code does
- How it works step by step
- Key design decisions
- Potential improvements`,
			Variables: []PromptVariable{
				{Name: "Target", Description: "The code to explain", Required: true},
			},
			Source: "builtin",
		},
		{
			Name:        "docs",
			Description: "Generate or update documentation",
			Tags:        []string{"documentation"},
			Complexity:  "low",
			Context:     []string{},
			Template: `Generate documentation for: {{.Target}}

Guidelines:
- Be clear and concise
- Include examples where helpful
- Document public APIs thoroughly
- Follow existing doc conventions`,
			Variables: []PromptVariable{
				{Name: "Target", Description: "The code or package to document", Required: true},
			},
			Source: "builtin",
		},
		{
			Name:        "deps",
			Description: "Update dependencies",
			Tags:        []string{"maintenance", "dependencies"},
			Complexity:  "low",
			Context:     []string{},
			Template: `Update dependencies for this project.

Guidelines:
- Check for outdated packages
- Update to latest compatible versions
- Note any breaking changes
- Run tests after updating`,
			Variables: []PromptVariable{},
			Source:    "builtin",
		},
		{
			Name:        "security",
			Description: "Security audit and fixes",
			Tags:        []string{"security", "audit"},
			Complexity:  "high",
			Context:     []string{},
			Template: `Perform a security audit of: {{.Target}}

Check for:
- Input validation issues
- Authentication/authorization gaps
- Sensitive data exposure
- Dependency vulnerabilities
- Common CWE patterns

Fix any issues found and explain the changes.`,
			Variables: []PromptVariable{
				{Name: "Target", Description: "The code or package to audit", Required: true},
			},
			Source: "builtin",
		},
		{
			Name:        "performance",
			Description: "Performance optimization",
			Tags:        []string{"performance", "optimization"},
			Complexity:  "high",
			Context:     []string{},
			Template: `Optimize performance for: {{.Target}}

Guidelines:
- Identify bottlenecks first
- Measure before and after
- Prefer algorithmic improvements over micro-optimizations
- Document trade-offs`,
			Variables: []PromptVariable{
				{Name: "Target", Description: "The code to optimize", Required: true},
			},
			Source: "builtin",
		},
	}
}
