package cli

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func newTestApp(t *testing.T, h Handlers) *App {
	t.Helper()
	app, err := New(Config{Handlers: h})
	require.NoError(t, err)
	return app
}
