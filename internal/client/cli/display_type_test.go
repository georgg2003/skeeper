package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/georgg2003/skeeper/internal/client/pkg/models"
)

func TestDisplayType(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{models.EntryTypePassword, "password"},
		{models.EntryTypeText, "text"},
		{models.EntryTypeFile, "file"},
		{models.EntryTypeCard, "card"},
		{"CUSTOM", "CUSTOM"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, displayType(tt.in), "displayType(%q)", tt.in)
	}
}
