package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/IAPro-Community/nexus/internal/control/computer"
	coreflags "github.com/IAPro-Community/nexus/internal/core/flags"
)

type computerStatus struct {
	FeatureEnabled bool                      `json:"feature_enabled"`
	Capabilities   computer.Capabilities     `json:"capabilities"`
	Permissions    computer.PermissionStatus `json:"permissions"`
}

func computerCmd(args []string) error {
	provider := computer.NewLocalProvider()
	jsonOutput := false
	filtered := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "--json" {
			jsonOutput = true
			continue
		}
		filtered = append(filtered, arg)
	}
	command := "status"
	if len(filtered) > 0 {
		command = strings.ToLower(strings.TrimSpace(filtered[0]))
	}
	ctx := context.Background()
	switch command {
	case "status":
		capabilities, err := provider.Capabilities(ctx)
		if err != nil {
			return err
		}
		permissions, err := provider.Permissions(ctx)
		if err != nil {
			return err
		}
		status := computerStatus{
			FeatureEnabled: coreflags.Enabled(coreflags.ComputerUse),
			Capabilities:   capabilities,
			Permissions:    permissions,
		}
		if jsonOutput {
			return json.NewEncoder(os.Stdout).Encode(status)
		}
		fmt.Printf("Computer use: enabled=%t provider=%s platform=%s available=%t\n",
			status.FeatureEnabled, status.Capabilities.Provider, status.Capabilities.Platform, status.Permissions.Available)
		if status.Permissions.Reason != "" {
			fmt.Printf("Reason: %s\n", status.Permissions.Reason)
		}
		return nil
	case "capabilities":
		capabilities, err := provider.Capabilities(ctx)
		if err != nil {
			return err
		}
		if jsonOutput {
			return json.NewEncoder(os.Stdout).Encode(capabilities)
		}
		fmt.Printf("%+v\n", capabilities)
		return nil
	case "permissions":
		permissions, err := provider.Permissions(ctx)
		if err != nil {
			return err
		}
		if jsonOutput {
			return json.NewEncoder(os.Stdout).Encode(permissions)
		}
		fmt.Printf("%+v\n", permissions)
		return nil
	default:
		return fmt.Errorf("usage: nexus computer [status|capabilities|permissions] [--json]")
	}
}
