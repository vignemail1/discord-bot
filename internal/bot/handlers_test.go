package bot_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"

	"github.com/vignemail1/discord-bot/internal/bot"
	"github.com/vignemail1/discord-bot/internal/cache"
	"github.com/vignemail1/discord-bot/internal/repository/mock"
)

func newTestHandler() (*mock.GuildRepositoryMock, *mock.ModuleRepositoryMock, *cache.GuildConfigCache, *bot.Handler) {
	gMock := mock.NewGuildRepository()
	mMock := mock.NewModuleRepository()
	cc := cache.New(mMock, 5*time.Minute)
	h := bot.NewHandler(gMock, mMock, cc)
	return gMock, mMock, cc, h
}

func TestHandleGuildCreate_OK(t *testing.T) {
	_, _, cc, h := newTestHandler()
	h.HandleGuildCreate(context.Background(), &discordgo.Guild{ID: "111", Name: "TestServer"})
	// Le cache doit être pré-populer (Populate ne retourne pas d'erreur).
	cfg, err := cc.Get(context.Background(), "111")
	assert.NoError(t, err)
	assert.Equal(t, "111", cfg.GuildID)
}

func TestHandleGuildCreate_UpsertError_NoPanic(t *testing.T) {
	gMock, _, _, h := newTestHandler()
	gMock.UpsertErr = errors.New("db error")
	// Ne doit pas paniquer
	h.HandleGuildCreate(context.Background(), &discordgo.Guild{ID: "222", Name: "Fail"})
}

func TestHandleGuildDelete_OK(t *testing.T) {
	gMock, mMock, cc, h := newTestHandler()
	_ = gMock.Upsert(context.Background(), mock.NewGuild("333", "ToDelete"))
	_ = mMock.Upsert(context.Background(), mock.NewModule("333", "invite_filter", true))
	_ = cc.Populate(context.Background(), "333")

	h.HandleGuildDelete(context.Background(), &discordgo.Guild{ID: "333"})
	// Après delete, l'entrée doit être invalide dans le cache.
	mMock.ListErr = errors.New("db down")
	_, err := cc.Get(context.Background(), "333")
	assert.Error(t, err, "cache should have been invalidated")
}

func TestHandleGuildDelete_DeactivateError_CacheStillInvalidated(t *testing.T) {
	gMock, mMock, cc, h := newTestHandler()
	_ = cc.Populate(context.Background(), "444")
	gMock.DeactivateErr = errors.New("db error")

	// Même si Deactivate échoue, le cache doit être invalidé.
	h.HandleGuildDelete(context.Background(), &discordgo.Guild{ID: "444"})
	mMock.ListErr = errors.New("db down")
	_, err := cc.Get(context.Background(), "444")
	assert.Error(t, err, "cache should still be invalidated despite DB error")
}
