package module_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vignemail1/discord-bot/internal/cache"
	"github.com/vignemail1/discord-bot/internal/module"
	modmock "github.com/vignemail1/discord-bot/internal/module/mock"
	"github.com/vignemail1/discord-bot/internal/repository"
	"github.com/vignemail1/discord-bot/internal/repository/mock"
)

// newTestDispatcher construit un Dispatcher avec un cache contenant les modules pré-configurés.
func newTestDispatcher(t *testing.T, modules []repository.GuildModule) (*module.Dispatcher, *module.Registry) {
	t.Helper()

	repoMock := mock.NewModuleRepository()
	for _, m := range modules {
		require.NoError(t, repoMock.Upsert(context.Background(), m))
	}

	cc := cache.New(repoMock, 5*time.Minute)
	reg := module.NewRegistry()
	disp := module.NewDispatcher(reg, cc)
	return disp, reg
}

func TestDispatcher_IgnoresBotMessages(t *testing.T) {
	disp, reg := newTestDispatcher(t, nil)
	var called atomic.Bool
	reg.MustRegister(module.NewHandlerFunc("mod", func(_ context.Context, _ *discordgo.Session, _ *discordgo.MessageCreate, _ *cache.GuildConfig) error {
		called.Store(true)
		return nil
	}))

	disp.OnMessageCreate(nil, &discordgo.MessageCreate{
		Message: &discordgo.Message{Author: &discordgo.User{Bot: true}, GuildID: "g1"},
	})
	assert.False(t, called.Load())
}

func TestDispatcher_IgnoresDMs(t *testing.T) {
	disp, reg := newTestDispatcher(t, nil)
	var called atomic.Bool
	reg.MustRegister(module.NewHandlerFunc("mod", func(_ context.Context, _ *discordgo.Session, _ *discordgo.MessageCreate, _ *cache.GuildConfig) error {
		called.Store(true)
		return nil
	}))

	disp.OnMessageCreate(nil, &discordgo.MessageCreate{
		Message: &discordgo.Message{Author: &discordgo.User{}, GuildID: ""}, // pas de GuildID
	})
	assert.False(t, called.Load())
}

func TestDispatcher_SkipsDisabledModule(t *testing.T) {
	disp, reg := newTestDispatcher(t, []repository.GuildModule{
		{GuildID: "g1", ModuleName: "mod_a", Enabled: false},
	})
	var called atomic.Bool
	reg.MustRegister(modmock.New("mod_a"))
	reg.All()[0].(*modmock.MockModule).OnHandle = func() { called.Store(true) }

	disp.OnMessageCreate(nil, &discordgo.MessageCreate{
		Message: &discordgo.Message{Author: &discordgo.User{}, GuildID: "g1"},
	})
	assert.False(t, called.Load(), "module désactivé ne doit pas être appelé")
}

func TestDispatcher_CallsEnabledModule(t *testing.T) {
	disp, reg := newTestDispatcher(t, []repository.GuildModule{
		{GuildID: "g1", ModuleName: "mod_a", Enabled: true},
	})
	var called atomic.Bool
	mod := modmock.New("mod_a")
	mod.OnHandle = func() { called.Store(true) }
	reg.MustRegister(mod)

	disp.OnMessageCreate(nil, &discordgo.MessageCreate{
		Message: &discordgo.Message{Author: &discordgo.User{}, GuildID: "g1"},
	})
	assert.True(t, called.Load(), "module actif doit être appelé")
}

func TestDispatcher_ContinuesOnModuleError(t *testing.T) {
	disp, reg := newTestDispatcher(t, []repository.GuildModule{
		{GuildID: "g1", ModuleName: "mod_err", Enabled: true},
		{GuildID: "g1", ModuleName: "mod_ok", Enabled: true},
	})
	var secondCalled atomic.Bool
	errMod := modmock.NewWithError("mod_err")
	okMod := modmock.New("mod_ok")
	okMod.OnHandle = func() { secondCalled.Store(true) }
	reg.MustRegister(errMod)
	reg.MustRegister(okMod)

	disp.OnMessageCreate(nil, &discordgo.MessageCreate{
		Message: &discordgo.Message{Author: &discordgo.User{}, GuildID: "g1"},
	})
	assert.True(t, secondCalled.Load(), "le deuxième module doit être appelé même si le premier échoue")
}
