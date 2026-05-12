package invitefilter_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vignemail1/discord-bot/internal/module/invitefilter"
)

// TestModule_Audit_DeleteRecorded vérifie qu'une suppression simple crée un enregistrement audit.
func TestModule_Audit_DeleteRecorded(t *testing.T) {
	counters := invitefilter.NewMemoryCounterRepo()
	audit := invitefilter.NewMemoryAuditRepo()
	mod := invitefilter.NewWithAudit(counters, audit)

	c := buildCache(t, "g1", invitefilter.Config{})
	cfgCache, _ := c.Get(context.Background(), "g1")

	msg := newMsg("g1", "u1", "discord.gg/badlink", nil)
	_ = mod.HandleMessage(context.Background(), nil, msg, cfgCache)

	records, err := audit.ListByUser(context.Background(), "g1", "u1", 10)
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, invitefilter.ActionDelete, records[0].Action)
	assert.Equal(t, 1, records[0].Count)
	assert.Equal(t, "badlink", records[0].InviteCodes)
	assert.Equal(t, "g1", records[0].GuildID)
	assert.Equal(t, "u1", records[0].UserID)
	assert.Equal(t, "chan1", records[0].ChannelID)
}

// TestModule_Audit_TimeoutRecorded vérifie qu'un timeout crée un enregistrement audit.
func TestModule_Audit_TimeoutRecorded(t *testing.T) {
	counters := invitefilter.NewMemoryCounterRepo()
	audit := invitefilter.NewMemoryAuditRepo()

	// Pré-charger 1 infraction pour déclencher le timeout au 2ème message.
	_, _ = counters.Increment(context.Background(), "g1", "u2", invitefilter.ModuleName)

	mod := invitefilter.NewWithAudit(counters, audit)
	c := buildCache(t, "g1", invitefilter.Config{})
	cfgCache, _ := c.Get(context.Background(), "g1")

	msg := newMsg("g1", "u2", "discord.gg/badlink", nil)
	_ = mod.HandleMessage(context.Background(), nil, msg, cfgCache)

	records, err := audit.ListByUser(context.Background(), "g1", "u2", 10)
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, invitefilter.ActionTimeout, records[0].Action)
	assert.Equal(t, 2, records[0].Count)
}

// TestModule_Audit_BanRecorded vérifie qu'un ban crée un enregistrement audit.
func TestModule_Audit_BanRecorded(t *testing.T) {
	counters := invitefilter.NewMemoryCounterRepo()
	audit := invitefilter.NewMemoryAuditRepo()

	// Pré-charger 2 infractions pour déclencher le ban au 3ème message.
	_, _ = counters.Increment(context.Background(), "g1", "u3", invitefilter.ModuleName)
	_, _ = counters.Increment(context.Background(), "g1", "u3", invitefilter.ModuleName)

	mod := invitefilter.NewWithAudit(counters, audit)
	c := buildCache(t, "g1", invitefilter.Config{BanThreshold: 3})
	cfgCache, _ := c.Get(context.Background(), "g1")

	msg := newMsg("g1", "u3", "discord.gg/badlink", nil)
	_ = mod.HandleMessage(context.Background(), nil, msg, cfgCache)

	records, err := audit.ListByUser(context.Background(), "g1", "u3", 10)
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, invitefilter.ActionBan, records[0].Action)
	assert.Equal(t, 3, records[0].Count)
}

// TestModule_Audit_NoAudit_NoRepo vérifie que l'absence d'AuditRepository ne panique pas.
func TestModule_Audit_NoRepo_NoPanic(t *testing.T) {
	counters := invitefilter.NewMemoryCounterRepo()
	mod := invitefilter.New(counters) // sans audit

	c := buildCache(t, "g1", invitefilter.Config{})
	cfgCache, _ := c.Get(context.Background(), "g1")

	msg := newMsg("g1", "u9", "discord.gg/norepo", nil)
	err := mod.HandleMessage(context.Background(), nil, msg, cfgCache)
	require.NoError(t, err) // pas de panique, pas d'erreur
}

// TestModule_Audit_InsertErr_DoesNotBlockAction vérifie qu'une erreur d'audit ne bloque pas la sanction.
func TestModule_Audit_InsertErr_DoesNotBlockAction(t *testing.T) {
	counters := invitefilter.NewMemoryCounterRepo()
	audit := invitefilter.NewMemoryAuditRepo()
	audit.InsertErr = assert.AnError

	mod := invitefilter.NewWithAudit(counters, audit)
	c := buildCache(t, "g1", invitefilter.Config{})
	cfgCache, _ := c.Get(context.Background(), "g1")

	msg := newMsg("g1", "u10", "discord.gg/failaudit", nil)
	err := mod.HandleMessage(context.Background(), nil, msg, cfgCache)
	// L'erreur d'audit est loguée mais ne remonte pas.
	require.NoError(t, err)

	// Le compteur a quand même été incrémenté.
	count, _ := counters.Get(context.Background(), "g1", "u10", invitefilter.ModuleName)
	assert.Equal(t, 1, count)
}

// TestModule_Audit_MultipleActions vérifie ListByGuild avec plusieurs utilisateurs.
func TestModule_Audit_MultipleActions_ListByGuild(t *testing.T) {
	counters := invitefilter.NewMemoryCounterRepo()
	audit := invitefilter.NewMemoryAuditRepo()
	mod := invitefilter.NewWithAudit(counters, audit)

	c := buildCache(t, "g1", invitefilter.Config{})
	cfgCache, _ := c.Get(context.Background(), "g1")

	for _, uid := range []string{"u1", "u2", "u3"} {
		msg := newMsg("g1", uid, "discord.gg/spam", nil)
		_ = mod.HandleMessage(context.Background(), nil, msg, cfgCache)
	}

	results, err := audit.ListByGuild(context.Background(), "g1", 10)
	require.NoError(t, err)
	assert.Len(t, results, 3)
}
