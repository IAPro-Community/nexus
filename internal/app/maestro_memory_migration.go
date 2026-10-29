package app

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/nexus"
)


var safeLegacyProjectID = regexp.MustCompile(`^prj_[A-Za-z0-9_-]{1,120}$`)

type maestroMemoryMigrationArgs struct {
	Command            string
	ProjectID          string
	ConfirmLocalBackup bool
	JSON               bool
}

func parseMaestroMemoryMigrationArgs(args []string) (maestroMemoryMigrationArgs, error) {
	var result maestroMemoryMigrationArgs
	if len(args) == 0 || (args[0] != "memory-preview" && args[0] != "memory-backup") {
		return result, errors.New("use maestro memory-preview or memory-backup")
	}
	result.Command = args[0]
	set := flag.NewFlagSet("maestro "+args[0], flag.ContinueOnError)
	set.SetOutput(io.Discard)
	set.StringVar(&result.ProjectID, "project-id", "", "Nexus project ID")
	set.BoolVar(&result.ConfirmLocalBackup, "confirm-local-backup", false, "write a private local backup (never imports data)")
	set.BoolVar(&result.JSON, "json", false, "machine-readable output")
	if err := set.Parse(args[1:]); err != nil {
		return maestroMemoryMigrationArgs{}, fmt.Errorf("invalid memory migration flags: %w", err)
	}
	if len(set.Args()) != 0 || strings.TrimSpace(result.ProjectID) == "" ||
		strings.TrimSpace(result.ProjectID) != result.ProjectID {
		return maestroMemoryMigrationArgs{}, errors.New("--project-id is required and positional args are not supported")
	}
	if !safeLegacyProjectID.MatchString(result.ProjectID) {
		return maestroMemoryMigrationArgs{}, errors.New("invalid Nexus project ID")
	}
	switch result.Command {
	case "memory-preview":
		if result.ConfirmLocalBackup {
			return maestroMemoryMigrationArgs{}, errors.New("memory-preview must not create backups")
		}
	case "memory-backup":
		if !result.ConfirmLocalBackup {
			return maestroMemoryMigrationArgs{}, errors.New("memory-backup requires --confirm-local-backup")
		}
	}
	return result, nil
}

// These commands are deliberately local CLI-only; they do not expose legacy
// values over a browser API and never import, delete or change memory.
// OpenStore is used instead of nexus.Default() to avoid initializing the
// persistent background Mission schedule loop for an inventory command.
func maestroMemoryMigrationCmd(args []string) error {
	parsed, err := parseMaestroMemoryMigrationArgs(args)
	if err != nil {
		return err
	}
	// Never create a new empty SQLite database merely because an operator
	// requested an inventory of existing memory.
	dataDir, err := config.DataDir()
	if err != nil {
		return err
	}
	database := filepath.Join(dataDir, "nexus.db")
	info, err := os.Lstat(database)
	if err != nil {
		return fmt.Errorf("existing Nexus project database required: %w", err)
	}
	if !info.Mode().IsRegular() {
		return errors.New("Nexus project database must be a regular file")
	}
	st, err := nexus.OpenStore()
	if err != nil {
		return fmt.Errorf("cannot inspect local Nexus store: %w", err)
	}
	defer st.Close()
	service := nexus.NewWithStoreAndExecutor(st, nil)
	var output any
	switch parsed.Command {
	case "memory-preview":
		output, err = service.PreviewLegacyIntelligenceMemoryMigration(context.Background(), parsed.ProjectID)
	case "memory-backup":
		output, err = service.BackupLegacyIntelligenceMemories(context.Background(), parsed.ProjectID)
	}
	if err != nil {
		return err
	}
	encoded, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return err
	}
	// JSON is always used to preserve stable fields in both machine and
	// operator modes. No raw memory values are ever written to stdout.
	fmt.Println(string(encoded))
	return nil
}
