package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/IAPro-Community/nexus/internal/nexus"
	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func planCmd(args []string) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "-h" {
		fmt.Println("Usage: nexus plan [list|create|show|compile|run] [args...]")
		return nil
	}

	n := nexus.Default()
	st, err := n.OpenProject()
	if err != nil {
		return err
	}

	switch args[0] {
	case "list", "ls":
		projects, err := st.ListProjects()
		if err != nil {
			return err
		}
		if len(projects) == 0 {
			fmt.Println("Nenhum projeto encontrado. Registre um projeto com: nexus projects add <caminho>")
			return nil
		}
		projID := projects[0].ID
		if len(args) > 1 {
			projID = args[1]
		}
		plans, err := n.ListWorkPlans(context.Background(), projID)
		if err != nil {
			return err
		}
		fmt.Printf("=== WorkPlans do Projeto (%s) ===\n", projID)
		for _, p := range plans {
			fmt.Printf("- [%s] %s (Rev %d, Status: %s, Fases: %d)\n", p.ID, p.Title, p.CurrentRevision, p.Status, len(p.Phases))
		}
		return nil

	case "create":
		if len(args) < 2 {
			return fmt.Errorf("informe o objetivo do plano, ex: nexus plan create \"Implementar auth JWT\"")
		}
		projects, err := st.ListProjects()
		if err != nil || len(projects) == 0 {
			return fmt.Errorf("nenhum projeto ativo; registre um projeto primeiro")
		}
		goal := strings.Join(args[1:], " ")
		//nolint:staticcheck // SA1019: retained solely for legacy `nexus plan create`.
		plan, err := n.GeneratePlanFromIntent(context.Background(), projects[0].ID, goal)
		if err != nil {
			return err
		}
		fmt.Printf("Plano criado com sucesso: %s [%s]\n", plan.Title, plan.ID)
		for _, ph := range plan.Phases {
			fmt.Printf("  Fase: %s (%d pacotes)\n", ph.Title, len(ph.Packages))
			for _, pkg := range ph.Packages {
				fmt.Printf("    * [%s] %s (%s)\n", pkg.ID, pkg.Title, pkg.Priority)
			}
		}
		return nil

	case "show":
		if len(args) < 2 {
			return fmt.Errorf("informe o ID do plano, ex: nexus plan show <plan-id>")
		}
		plan, err := n.GetWorkPlan(context.Background(), args[1])
		if err != nil {
			return err
		}
		out, _ := json.MarshalIndent(plan, "", "  ")
		fmt.Println(string(out))
		return nil

	case "compile":
		if len(args) < 3 {
			return fmt.Errorf("informe o ID do plano e do pacote, ex: nexus plan compile <plan-id> <package-id> [phase-id]")
		}
		phaseID := ""
		if len(args) > 3 {
			phaseID = args[3]
		}
		compiled, err := n.CompilePackagePrompt(context.Background(), args[1], phaseID, args[2])
		if err != nil {
			return err
		}
		out, _ := json.MarshalIndent(compiled, "", "  ")
		fmt.Println(string(out))
		return nil

	case "run":
		if len(args) < 2 {
			return fmt.Errorf("informe o ID do plano, ex: nexus plan run <plan-id> [agent-id]")
		}
		defaultAgentID := ""
		if len(args) > 2 {
			defaultAgentID = args[2]
		}
		// Same public contract as Web POST /plans/{id}/run and nexus run "<goal>":
		// autonomous worker owns progress; CLI must not sync-abort on first step.
		run, err := n.StartMissionRun(context.Background(), args[1], defaultAgentID, runner.DefaultAutonomyContract(), true)
		if err != nil {
			return err
		}
		return waitAndPrintMission(context.Background(), n, run, map[string]any{
			"autonomous": true,
			"plan_id":    args[1],
		})

	default:
		return fmt.Errorf("subcomando de plano desconhecido: %s", args[0])
	}
}

func waitAndPrintMission(ctx context.Context, n *nexus.Nexus, started *runner.MissionRun, fields map[string]any) error {
	if started == nil {
		return fmt.Errorf("mission start returned no run")
	}
	final, waitErr := n.WaitMissionWorker(ctx, started.ID)
	if final == nil {
		final = started
	}
	result := map[string]any{
		"run_id": final.ID,
		"state":  final.State,
	}
	for key, value := range fields {
		result[key] = value
	}
	if final.CompletedAt != nil {
		result["completed_at"] = final.CompletedAt
	}
	out, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(out))
	return waitErr
}

func agentsCmd(args []string) error {
	n := nexus.Default()
	st, err := n.OpenProject()
	if err != nil {
		return err
	}

	projects, err := st.ListProjects()
	if err != nil || len(projects) == 0 {
		fmt.Println("Nenhum projeto encontrado.")
		return nil
	}

	projID := projects[0].ID
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		projID = args[0]
	}

	agents, err := st.ListAgents(projID)
	if err != nil {
		return err
	}

	fmt.Printf("=== Agentes Nexus (%s) ===\n", projID)
	for _, a := range agents {
		eff, _ := n.EffectiveAgentState(a.ID)
		fmt.Printf("- [%s] %s (Papel: %s, Status Real: %s, Revisão: %s)\n",
			a.ID, a.Name, a.Role, eff, a.CurrentRevisionID)
	}
	return nil
}

func projectsCmd(args []string) error {
	n := nexus.Default()
	st, err := n.OpenProject()
	if err != nil {
		return err
	}

	if len(args) == 0 || args[0] == "list" || args[0] == "ls" {
		projects, err := st.ListProjects()
		if err != nil {
			return err
		}
		fmt.Println("=== Projetos Registrados no Nexus ===")
		for _, p := range projects {
			fmt.Printf("- [%s] %s (%s, Branch: %s, Isolation: %s)\n", p.ID, p.Name, p.CanonicalPath, p.DefaultBranch, p.DefaultIsolation)
		}
		return nil
	}

	if args[0] == "add" {
		opts, err := parseProjectAddArgs(args[1:])
		if err != nil {
			return err
		}
		proj, err := st.CreateProject(store.Project{
			CanonicalPath:    opts.Path,
			Name:             opts.Name,
			DefaultIsolation: opts.Isolation,
		})
		if err != nil {
			return err
		}
		fmt.Printf("Projeto registrado com sucesso: %s [%s] (isolation=%s)\n", proj.Name, proj.ID, proj.DefaultIsolation)
		return nil
	}

	return fmt.Errorf("subcomando de projeto desconhecido: %s", args[0])
}

type projectAddOptions struct {
	Path      string
	Name      string
	Isolation string
}

func parseProjectAddArgs(args []string) (projectAddOptions, error) {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		return projectAddOptions{}, fmt.Errorf("informe o caminho do projeto, ex: nexus projects add /caminho/do/projeto --isolation worktree")
	}
	opts := projectAddOptions{Path: args[0]}
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--name":
			if i+1 >= len(args) || strings.TrimSpace(args[i+1]) == "" {
				return projectAddOptions{}, fmt.Errorf("--name requer um valor")
			}
			if opts.Name != "" {
				return projectAddOptions{}, fmt.Errorf("nome do projeto informado mais de uma vez")
			}
			opts.Name = args[i+1]
			i++
		case "--isolation":
			if i+1 >= len(args) {
				return projectAddOptions{}, fmt.Errorf("--isolation requer project, worktree ou none")
			}
			opts.Isolation = strings.ToLower(strings.TrimSpace(args[i+1]))
			i++
			switch opts.Isolation {
			case "project", "worktree", "none":
			default:
				return projectAddOptions{}, fmt.Errorf("--isolation inválido %q; use project, worktree ou none", opts.Isolation)
			}
		default:
			// Preserve the historical positional display name while preventing
			// ambiguous extra positional arguments.
			if strings.HasPrefix(args[i], "-") {
				return projectAddOptions{}, fmt.Errorf("opção desconhecida: %s", args[i])
			}
			if opts.Name != "" {
				return projectAddOptions{}, fmt.Errorf("nome do projeto informado mais de uma vez")
			}
			opts.Name = args[i]
		}
	}
	return opts, nil
}

func conductorCmd(args []string) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "-h" {
		fmt.Println("Usage: nexus conductor [policy|dispatch] [args...]")
		fmt.Println("  policy get           Show current conductor policy")
		fmt.Println("  policy set <json>    Update conductor policy")
		fmt.Println("  dispatch <goal>      Dispatch a goal through the conductor")
		fmt.Println("  --json               Output as JSON")
		return nil
	}

	jsonOutput := false
	cleanArgs := []string{}
	for _, a := range args {
		if a == "--json" {
			jsonOutput = true
		} else {
			cleanArgs = append(cleanArgs, a)
		}
	}
	args = cleanArgs

	n := nexus.Default()
	st, err := n.OpenProject()
	if err != nil {
		return err
	}

	switch args[0] {
	case "policy":
		if len(args) < 2 {
			return fmt.Errorf("usage: nexus conductor policy [get|set]")
		}
		switch args[1] {
		case "get":
			policy, err := nexus.GetConductorPolicy(st)
			if err != nil {
				return err
			}
			if jsonOutput {
				out, _ := json.MarshalIndent(policy, "", "  ")
				fmt.Println(string(out))
			} else {
				fmt.Printf("Conductor Policy (v%d):\n", policy.Version)
				if policy.ReviewMode != nil {
					fmt.Printf("  ReviewMode: %s\n", *policy.ReviewMode)
				}
				if policy.RetentionDays != nil {
					fmt.Printf("  RetentionDays: %d\n", *policy.RetentionDays)
				}
				if policy.IntegrationOptIn != nil {
					fmt.Printf("  IntegrationOptIn: %v\n", *policy.IntegrationOptIn)
				}
			}
			return nil
		case "set":
			if len(args) < 3 {
				return fmt.Errorf("usage: nexus conductor policy set '<json>'")
			}
			var policy nexus.ConductorPolicy
			if err := json.Unmarshal([]byte(args[2]), &policy); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}
			if err := nexus.PutConductorPolicy(st, &policy); err != nil {
				return err
			}
			fmt.Println("Policy updated successfully.")
			return nil
		default:
			return fmt.Errorf("unknown policy subcommand: %s", args[1])
		}

	case "dispatch":
		if len(args) < 2 {
			return fmt.Errorf("usage: nexus conductor dispatch <goal>")
		}
		goal := strings.Join(args[1:], " ")
		projects, err := st.ListProjects()
		if err != nil || len(projects) == 0 {
			return fmt.Errorf("no project found")
		}
		result, err := n.Dispatch(context.Background(), nexus.DispatchRequest{
			ProjectID:      projects[0].ID,
			Goal:           goal,
			SourcePrompt:   goal,
			DelegationMode: nexus.DelegationAuto,
		})
		if err != nil {
			return err
		}
		if jsonOutput {
			out, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(out))
		} else {
			fmt.Printf("Dispatch result: %s (Run: %s)\n", result.Status, result.RunID)
			if result.Error != "" {
				fmt.Printf("Error: %s\n", result.Error)
			}
		}
		return nil

	default:
		return fmt.Errorf("unknown conductor subcommand: %s", args[0])
	}
}
