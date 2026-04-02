package prompt

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuiltinPrompts(t *testing.T) {
	t.Parallel()

	prompts := BuiltinPrompts()
	assert.Equal(t, 10, len(prompts))

	lib := NewLibrary()
	for _, p := range prompts {
		err := lib.Add(p)
		require.NoError(t, err, "failed to add prompt: %s", p.Name)
	}

	assert.Equal(t, 10, lib.Count())
}

func TestBuiltinPromptNames(t *testing.T) {
	t.Parallel()

	expected := []string{
		"add-feature", "fix-bug", "refactor", "write-tests",
		"review", "explain", "docs", "deps", "security", "performance",
	}

	prompts := BuiltinPrompts()
	lib := NewLibrary()
	for _, p := range prompts {
		require.NoError(t, lib.Add(p))
	}

	for _, name := range expected {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			p, ok := lib.Get(name)
			require.True(t, ok, "missing builtin prompt: %s", name)
			assert.Equal(t, "builtin", p.Source)
			assert.NotEmpty(t, p.Description)
			assert.NotEmpty(t, p.Template)
		})
	}
}

func TestPromptTemplateValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		tmpl    *PromptTemplate
		wantErr bool
	}{
		{
			name: "valid template",
			tmpl: &PromptTemplate{
				Name:     "test",
				Template: "Hello {{.Name}}",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			tmpl: &PromptTemplate{
				Template: "Hello",
			},
			wantErr: true,
		},
		{
			name: "missing template",
			tmpl: &PromptTemplate{
				Name: "test",
			},
			wantErr: true,
		},
		{
			name: "invalid template syntax",
			tmpl: &PromptTemplate{
				Name:     "test",
				Template: "Hello {{.Name",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.tmpl.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPromptTemplateRender(t *testing.T) {
	t.Parallel()

	tmpl := &PromptTemplate{
		Name:     "greeting",
		Template: "Hello {{.Name}}, welcome to {{.Project}}!",
		Variables: []PromptVariable{
			{Name: "Name", Description: "User name", Required: true},
			{Name: "Project", Description: "Project name", Required: true},
		},
	}

	result, err := tmpl.Render(map[string]string{
		"Name":    "Alice",
		"Project": "HELM",
	})
	require.NoError(t, err)
	assert.Equal(t, "Hello Alice, welcome to HELM!", result)
}

func TestPromptTemplateRenderMissingRequired(t *testing.T) {
	t.Parallel()

	tmpl := &PromptTemplate{
		Name:     "greeting",
		Template: "Hello {{.Name}}!",
		Variables: []PromptVariable{
			{Name: "Name", Description: "User name", Required: true},
		},
	}

	_, err := tmpl.Render(map[string]string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "required variable")
}

func TestPromptTemplateRenderWithDefault(t *testing.T) {
	t.Parallel()

	tmpl := &PromptTemplate{
		Name:     "greeting",
		Template: "Hello {{.Name}}!",
		Variables: []PromptVariable{
			{Name: "Name", Description: "User name", Required: false, Default: "World"},
		},
	}

	result, err := tmpl.Render(map[string]string{})
	require.NoError(t, err)
	assert.Equal(t, "Hello World!", result)
}

func TestPromptLibrarySearch(t *testing.T) {
	t.Parallel()

	lib := NewLibrary()
	for _, p := range BuiltinPrompts() {
		require.NoError(t, lib.Add(p))
	}

	results := lib.Search("test")
	assert.GreaterOrEqual(t, len(results), 1)

	results = lib.Search("")
	assert.Equal(t, 10, len(results))
}

func TestPromptLibraryByTag(t *testing.T) {
	t.Parallel()

	lib := NewLibrary()
	for _, p := range BuiltinPrompts() {
		require.NoError(t, lib.Add(p))
	}

	results := lib.ByTag("security")
	assert.Equal(t, 1, len(results))
	assert.Equal(t, "security", results[0].Name)
}

func TestPromptLibraryTags(t *testing.T) {
	t.Parallel()

	lib := NewLibrary()
	for _, p := range BuiltinPrompts() {
		require.NoError(t, lib.Add(p))
	}

	tags := lib.Tags()
	assert.GreaterOrEqual(t, len(tags), 5)
	assert.Contains(t, tags, "feature")
	assert.Contains(t, tags, "testing")
	assert.Contains(t, tags, "security")
}

func TestPromptTemplateRequiredVariables(t *testing.T) {
	t.Parallel()

	tmpl := &PromptTemplate{
		Name:     "test",
		Template: "Hello {{.Name}}",
		Variables: []PromptVariable{
			{Name: "Name", Description: "Name", Required: true},
			{Name: "Greeting", Description: "Greeting", Required: false, Default: "Hi"},
		},
	}

	req := tmpl.RequiredVariables()
	assert.Equal(t, 1, len(req))
	assert.Equal(t, "Name", req[0].Name)
}

func TestPromptTemplateHasTag(t *testing.T) {
	t.Parallel()

	tmpl := &PromptTemplate{
		Name:     "test",
		Template: "Hello",
		Tags:     []string{"feature", "implementation"},
	}

	assert.True(t, tmpl.HasTag("feature"))
	assert.False(t, tmpl.HasTag("unknown"))
}
