package wakeup

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWakeupCommand(t *testing.T) {
	cmd := NewWakeupCommand()

	require.NotNil(t, cmd)

	assert.Equal(t, "wakeup", cmd.Use)
	assert.Equal(t, "Initialize Spiderweb configuration and workspace", cmd.Short)

	assert.Len(t, cmd.Aliases, 2)
	assert.True(t, cmd.HasAlias("wake"))
	assert.True(t, cmd.HasAlias("o"))

	assert.NotNil(t, cmd.Run)
	assert.Nil(t, cmd.RunE)

	assert.Nil(t, cmd.PersistentPreRun)
	assert.Nil(t, cmd.PersistentPostRun)

	assert.False(t, cmd.HasFlags())
	assert.False(t, cmd.HasSubCommands())
}
