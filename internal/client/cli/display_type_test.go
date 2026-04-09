package cli

import (
	"testing"

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
		if got := displayType(tt.in); got != tt.want {
			t.Fatalf("displayType(%q)=%q want %q", tt.in, got, tt.want)
		}
	}
}
