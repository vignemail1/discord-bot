package module_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vignemail1/discord-bot/internal/module"
	modmock "github.com/vignemail1/discord-bot/internal/module/mock"
)

func TestRegistry_Register_OK(t *testing.T) {
	reg := module.NewRegistry()
	err := reg.Register(modmock.New("mod_a"))
	require.NoError(t, err)
	assert.Equal(t, []string{"mod_a"}, reg.Names())
}

func TestRegistry_Register_Duplicate(t *testing.T) {
	reg := module.NewRegistry()
	require.NoError(t, reg.Register(modmock.New("mod_a")))
	err := reg.Register(modmock.New("mod_a"))
	require.Error(t, err)
}

func TestRegistry_MustRegister_Panics(t *testing.T) {
	reg := module.NewRegistry()
	reg.MustRegister(modmock.New("mod_a"))
	assert.Panics(t, func() { reg.MustRegister(modmock.New("mod_a")) })
}

func TestRegistry_All_Order(t *testing.T) {
	reg := module.NewRegistry()
	for _, name := range []string{"c", "a", "b"} {
		require.NoError(t, reg.Register(modmock.New(name)))
	}
	names := make([]string, 0, 3)
	for _, m := range reg.All() {
		names = append(names, m.Name())
	}
	assert.Equal(t, []string{"c", "a", "b"}, names, "ordre d'enregistrement conservé")
}

func TestRegistry_Get(t *testing.T) {
	reg := module.NewRegistry()
	reg.MustRegister(modmock.New("mod_a"))

	m, ok := reg.Get("mod_a")
	assert.True(t, ok)
	assert.Equal(t, "mod_a", m.Name())

	_, ok = reg.Get("unknown")
	assert.False(t, ok)
}
