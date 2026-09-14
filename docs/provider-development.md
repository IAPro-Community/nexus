# Provider Adapter Development Guide

**🇬🇧 English** | [🇧🇷 Português](provider-development.pt-BR.md) | [🇪🇸 Español](provider-development.es.md)

This guide explains how to add support for a new AI Coding CLI to `ai-cli` without modifying the core control plane.

---

## 1. Provider Interfaces

Every provider adapter lives in `internal/core/provider/adapters/<name>/` and implements `provider.Provider`.

```go
type Provider interface {
    ID() model.ProviderID
    Name() string
    Detect(ctx context.Context) model.DetectionResult
    Capabilities() model.Capabilities
    Prepare(ctx context.Context, p model.Profile) error
    Run(ctx context.Context, p model.Profile, args []string) (model.Failure, error)
}
```

### Optional Capabilities Interfaces:
- **`AuthProvider`**: Implements `Login`, `Logout`, `InspectAuth`.
- **`UsageProvider`**: Implements `GetUsage`.
- **`ConversationProvider`**: Implements `ListConversations`, `Resume`.
- **`ErrorClassifier`**: Implements `ClassifyError(err error, output string) model.Failure`.

---

## 2. Step-by-Step Implementation

### Step 1: Create the Adapter Package
Create `internal/core/provider/adapters/myagent/myagent.go`.

```go
package myagent

import (
    "context"
    "os"
    "path/filepath"
    "strings"
    
    "github.com/IAPro-Community/nexus/internal/core/config"
    "github.com/IAPro-Community/nexus/internal/core/model"
    "github.com/IAPro-Community/nexus/internal/core/security"
    "github.com/IAPro-Community/nexus/internal/runtime"
)

type Adapter struct{}

func New() *Adapter { return &Adapter{} }

func (a *Adapter) ID() model.ProviderID { return "myagent" }
func (a *Adapter) Name() string         { return "MyAgent CLI" }

func (a *Adapter) Capabilities() model.Capabilities {
    return model.Capabilities{
        Login:           true,
        Logout:          true,
        Usage:           true,
        Conversations:   true,
        Resume:          true,
        IsolatedRuntime: true,
        ProjectBinding:  true,
    }
}
```

### Step 2: Implement Detection & Preparation
```go
func (a *Adapter) Detect(ctx context.Context) model.DetectionResult {
    bin, err := runtime.LookPath("myagent")
    if err != nil {
        return model.DetectionResult{Installed: false, Error: "not found in PATH"}
    }
    out, _ := runtime.RunCommandCapture(ctx, bin, []string{"--version"}, os.Environ(), "")
    return model.DetectionResult{Installed: true, BinaryPath: bin, Version: strings.TrimSpace(out)}
}

func (a *Adapter) Prepare(ctx context.Context, p model.Profile) error {
    home, err := config.ProfileHome(string(a.ID()), p.Name)
    if err != nil {
        return err
    }
    _ = os.MkdirAll(home, 0700)
    cfgObj, _ := config.LoadConfig()
    return security.ApplyIsolation(home, security.GetPolicy(cfgObj.IsolationPreset))
}
```

### Step 3: Implement Interactive Process Execution
```go
func (a *Adapter) Run(ctx context.Context, p model.Profile, args []string) (model.Failure, error) {
    if err := a.Prepare(ctx, p); err != nil {
        return model.Failure{Kind: model.FailureCommand, Message: err.Error()}, err
    }
    bin, err := runtime.LookPath("myagent")
    if err != nil {
        return model.Failure{Kind: model.FailureProvider, Message: "myagent not found"}, err
    }
    home, _ := config.ProfileHome(string(a.ID()), p.Name)
    cwd, _ := os.Getwd()
    env := runtime.EnvSet(os.Environ(), map[string]string{
        "HOME": home,
        "MYAGENT_CONFIG_DIR": filepath.Join(home, ".config", "myagent"),
    })
    return runtime.RunInteractive(bin, args, env, cwd)
}
```

### Step 4: Register in `internal/app/app.go`
```go
reg := provider.NewRegistry()
_ = reg.Register(codex.New())
_ = reg.Register(agy.New())
_ = reg.Register(claude.New())
_ = reg.Register(opencode.New())
_ = reg.Register(gemini.New())
_ = reg.Register(myagent.New()) // Add your new adapter here
```

### Step 5: Run Tests
```bash
go test -v ./...
go test -race ./...
```
