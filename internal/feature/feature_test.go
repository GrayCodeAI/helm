package feature

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterAndIsEnabled(t *testing.T) {
	t.Parallel()

	m := NewManager()

	flag := Flag{
		Name:        "test_flag",
		Enabled:     true,
		Description: "Test flag",
	}
	m.Register(flag)

	assert.True(t, m.IsEnabled("test_flag", nil))
	assert.False(t, m.IsEnabled("nonexistent", nil))
}

func TestDisableAndEnable(t *testing.T) {
	t.Parallel()

	m := NewManager()
	m.Register(Flag{Name: "test_flag", Enabled: true})

	err := m.Disable("test_flag")
	require.NoError(t, err)
	assert.False(t, m.IsEnabled("test_flag", nil))

	err = m.Enable("test_flag")
	require.NoError(t, err)
	assert.True(t, m.IsEnabled("test_flag", nil))
}

func TestDisableNonexistent(t *testing.T) {
	t.Parallel()

	m := NewManager()
	err := m.Disable("nonexistent")
	assert.Error(t, err)
}

func TestGetVariant(t *testing.T) {
	t.Parallel()

	m := NewManager()
	m.Register(Flag{
		Name:    "test_flag",
		Enabled: true,
		Variants: map[string]interface{}{
			"default": "v1",
			"test":    "v2",
		},
	})

	variant := m.GetVariant("test_flag", nil)
	assert.Equal(t, "v1", variant)
}

func TestGetVariantDisabled(t *testing.T) {
	t.Parallel()

	m := NewManager()
	m.Register(Flag{
		Name:    "test_flag",
		Enabled: false,
		Variants: map[string]interface{}{
			"default": "v1",
		},
	})

	variant := m.GetVariant("test_flag", nil)
	assert.Nil(t, variant)
}

func TestList(t *testing.T) {
	t.Parallel()

	m := NewManager()
	m.Register(Flag{Name: "flag1", Enabled: true})
	m.Register(Flag{Name: "flag2", Enabled: false})

	flags := m.List()
	assert.Len(t, flags, 2)
}

func TestExportImport(t *testing.T) {
	t.Parallel()

	m := NewManager()
	m.Register(Flag{Name: "flag1", Enabled: true})
	m.Register(Flag{Name: "flag2", Enabled: false})

	data, err := m.Export()
	require.NoError(t, err)

	m2 := NewManager()
	err = m2.Import(data)
	require.NoError(t, err)

	assert.True(t, m2.IsEnabled("flag1", nil))
	assert.False(t, m2.IsEnabled("flag2", nil))
}

func TestSaveLoadFile(t *testing.T) {
	t.Parallel()

	m := NewManager()
	m.Register(Flag{Name: "flag1", Enabled: true})

	path := t.TempDir() + "/flags.json"
	err := m.SaveToFile(path)
	require.NoError(t, err)

	m2 := NewManager()
	err = m2.LoadFromFile(path)
	require.NoError(t, err)

	assert.True(t, m2.IsEnabled("flag1", nil))
}

func TestOnChange(t *testing.T) {
	t.Parallel()

	m := NewManager()
	m.Register(Flag{Name: "flag1", Enabled: true})

	called := false
	m.OnChange(func() {
		called = true
	})

	m.Enable("flag1")
	assert.True(t, called)
}

func TestCheck(t *testing.T) {
	t.Parallel()

	m := NewManager()
	m.Register(Flag{Name: "flag1", Enabled: true})

	assert.True(t, m.Check("flag1"))

	assert.Panics(t, func() {
		m.Check("nonexistent")
	})
}

func TestEvaluateRule(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		rule     Rule
		attrs    map[string]interface{}
		expected bool
	}{
		{"equals true", Rule{Attribute: "env", Operator: "equals", Value: "prod"}, map[string]interface{}{"env": "prod"}, true},
		{"equals false", Rule{Attribute: "env", Operator: "equals", Value: "prod"}, map[string]interface{}{"env": "dev"}, false},
		{"not_equals true", Rule{Attribute: "env", Operator: "not_equals", Value: "prod"}, map[string]interface{}{"env": "dev"}, true},
		{"missing attr", Rule{Attribute: "env", Operator: "equals", Value: "prod"}, map[string]interface{}{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := evaluateRule(tt.rule, tt.attrs)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDefaultFlags(t *testing.T) {
	t.Parallel()

	flags := DefaultFlags()
	assert.Len(t, flags, 4)

	m := NewManager()
	for _, f := range flags {
		m.Register(f)
	}

	assert.True(t, m.IsEnabled("web_dashboard", nil))
	assert.False(t, m.IsEnabled("voice_notes", nil))
	assert.False(t, m.IsEnabled("team_sync", nil))
	assert.True(t, m.IsEnabled("advanced_analytics", nil))
}

func TestFlagTimestamps(t *testing.T) {
	t.Parallel()

	m := NewManager()
	before := time.Now()
	m.Register(Flag{Name: "flag1", Enabled: true})
	after := time.Now()

	flags := m.List()
	require.Len(t, flags, 1)

	assert.True(t, flags[0].CreatedAt.After(before) || flags[0].CreatedAt.Equal(before))
	assert.True(t, flags[0].CreatedAt.Before(after) || flags[0].CreatedAt.Equal(after))
	assert.True(t, flags[0].UpdatedAt.After(before) || flags[0].UpdatedAt.Equal(before))
}

func TestContains(t *testing.T) {
	t.Parallel()

	assert.True(t, contains("hello world", "world"))
	assert.False(t, contains("hello", "world"))
	assert.True(t, contains("", ""))
}
