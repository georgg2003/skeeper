package delivery

import "github.com/georgg2003/skeeper/internal/client/pkg/models"

// DisplayType maps entry type codes to short labels for list output.
func DisplayType(t string) string {
	switch t {
	case models.EntryTypePassword:
		return "password"
	case models.EntryTypeText:
		return "text"
	case models.EntryTypeFile:
		return "file"
	case models.EntryTypeCard:
		return "card"
	default:
		return t
	}
}
