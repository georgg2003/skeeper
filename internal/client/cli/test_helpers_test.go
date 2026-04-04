package cli

import (
	"sync"
	"testing"
)

func resetCLIForTest(t *testing.T) {
	t.Helper()
	setupOnce = sync.Once{}
	setupErr = nil
	SetUseCases(nil, nil, nil)
	skipBootstrapForTest = true
	t.Cleanup(func() {
		skipBootstrapForTest = false
		setupOnce = sync.Once{}
		setupErr = nil
		SetUseCases(nil, nil, nil)
	})
}
