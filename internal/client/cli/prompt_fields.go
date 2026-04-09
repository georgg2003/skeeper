package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/georgg2003/skeeper/internal/client/pkg/models"
	"github.com/georgg2003/skeeper/internal/client/usecase"
)

func requireSecretUC() error {
	if secretUC == nil {
		return fmt.Errorf("client not initialized")
	}
	return nil
}

func requireSyncUC() error {
	if syncUC == nil {
		return fmt.Errorf("client not initialized")
	}
	return nil
}

func requireAuthUC() error {
	if authUC == nil {
		return fmt.Errorf("client not initialized")
	}
	return nil
}

func parseUUIDArg(s string) (uuid.UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid uuid: %w", err)
	}
	return id, nil
}

// promptEntryMetadata asks for entry title and optional notes (shared by add-* and update *).
func promptEntryMetadata(cmd *cobra.Command, namePrompt string) (usecase.EntryMetadata, error) {
	if namePrompt == "" {
		namePrompt = "Entry name: "
	}
	writePrompt(cmd, "%s", namePrompt)
	name, err := readLine(cmd)
	if err != nil {
		return usecase.EntryMetadata{}, err
	}
	writePrompt(cmd, "Notes (optional): ")
	notes, _ := readLine(cmd)
	return usecase.EntryMetadata{Name: name, Notes: notes}, nil
}

func promptMasterPassword(cmd *cobra.Command) (string, error) {
	writePrompt(cmd, "Master password: ")
	return readPasswordLine(cmd)
}

// promptPasswordValue reads one sensitive line after the given prompt (stored password, CVC, etc.).
func promptPasswordValue(cmd *cobra.Command, prompt string) (string, error) {
	writePrompt(cmd, "%s", prompt)
	return readPasswordLine(cmd)
}

// promptMultilineText reads lines until an empty line; trims a single trailing newline from the result.
func promptMultilineText(cmd *cobra.Command, banner string) (string, error) {
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), banner)
	var b strings.Builder
	for {
		line, err := readLine(cmd)
		if err != nil {
			return "", err
		}
		if line == "" {
			break
		}
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(line)
	}
	return strings.TrimSuffix(b.String(), "\n"), nil
}

// promptCard reads cardholder, number, expiry, and CVC.
func promptCard(cmd *cobra.Command) (models.CardPayload, error) {
	card := models.CardPayload{}
	var err error
	writePrompt(cmd, "Cardholder: ")
	card.Holder, err = readLine(cmd)
	if err != nil {
		return card, err
	}
	writePrompt(cmd, "Number: ")
	card.Number, err = readLine(cmd)
	if err != nil {
		return card, err
	}
	writePrompt(cmd, "Expiry (MM/YY): ")
	card.Expiry, err = readLine(cmd)
	if err != nil {
		return card, err
	}
	writePrompt(cmd, "CVC: ")
	card.CVC, err = readPasswordLine(cmd)
	return card, err
}

// promptOptionalFilePath reads a path line; empty trimmed path means do not replace file content.
func promptOptionalFilePath(cmd *cobra.Command, pathPrompt string) (data []byte, orig string, replace bool, err error) {
	writePrompt(cmd, "%s", pathPrompt)
	path, err := readLine(cmd)
	if err != nil {
		return nil, "", false, err
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, "", false, nil
	}
	data, err = os.ReadFile(path)
	if err != nil {
		return nil, "", false, fmt.Errorf("read file: %w", err)
	}
	return data, filepath.Base(path), true, nil
}
