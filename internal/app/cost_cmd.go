package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/IAPro-Community/nexus/internal/nexus"
)

const costSummaryUsage = "Usage: nexus cost summary [--days N] [--project-id <id>] [--json]"

// costCmd is the entry point of `nexus cost`. It is wiring only: every
// formula comes from the CostApplicationService / canonical store aggregates,
// never duplicated here. All subcommands are read-only over the local cost
// observation ledger.
func costCmd(args []string) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		fmt.Println(costSummaryUsage)
		fmt.Println("  Resumo local de custo sobre o ledger canônico de observações (somente leitura).")
		return nil
	}
	switch args[0] {
	case "summary":
		return costSummaryCmd(args[1:])
	default:
		return fmt.Errorf("unknown cost subcommand %q; %s", args[0], costSummaryUsage)
	}
}

// costSummaryCmd renders `nexus cost summary`. With --json it emits exactly
// the same schema as GET /api/v1/cost/analytics (CostAnalyticsReport), so
// CLI, web and TUI surfaces share one fixture. Days validation mirrors the
// HTTP handler: 1..365, default 7.
func costSummaryCmd(args []string) error {
	const (
		defaultDays   = 7
		minWindowDays = 1
		maxWindowDays = 365
	)
	days := defaultDays
	projectID := ""
	isJSON := false
	parseDays := func(raw string) error {
		value, err := strconv.Atoi(strings.TrimSpace(raw))
		if err != nil || value < minWindowDays || value > maxWindowDays {
			// Same message and boundaries as GET /api/v1/cost/analytics.
			return fmt.Errorf("days must be between 1 and 365")
		}
		days = value
		return nil
	}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--json":
			isJSON = true
		case arg == "--days":
			i++
			if i >= len(args) {
				return fmt.Errorf("--days requires a value; %s", costSummaryUsage)
			}
			if err := parseDays(args[i]); err != nil {
				return err
			}
		case strings.HasPrefix(arg, "--days="):
			if err := parseDays(strings.TrimPrefix(arg, "--days=")); err != nil {
				return err
			}
		case arg == "--project-id":
			i++
			if i >= len(args) {
				return fmt.Errorf("--project-id requires a value; %s", costSummaryUsage)
			}
			projectID = strings.TrimSpace(args[i])
		case strings.HasPrefix(arg, "--project-id="):
			projectID = strings.TrimSpace(strings.TrimPrefix(arg, "--project-id="))
		default:
			return fmt.Errorf("unknown flag %q; %s", arg, costSummaryUsage)
		}
	}

	svc := nexus.NewCostApplicationService(nexus.Default().Store())
	report, err := svc.Analytics(context.Background(), projectID, days)
	if err != nil {
		return err
	}

	if isJSON {
		b, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(b))
		return nil
	}

	total := report.Total
	fmt.Printf("=== Resumo de custo (%d dias) ===\n", report.Days)
	if report.ProjectID != "" {
		fmt.Printf("Projeto:       %s\n", report.ProjectID)
	}
	fmt.Printf("Fonte:         %s\n", report.Source)
	fmt.Printf("Desde:         %s\n", report.Since.Format("2006-01-02 15:04 UTC"))
	fmt.Printf("Observações:   %d (falhas: %d)\n", total.ObservationCount, total.InvocationsFailed)
	if total.ErrorRate != nil {
		fmt.Printf("Taxa de erro:  %.1f%%\n", *total.ErrorRate*100)
	}
	if total.AvgDurationMS != nil {
		fmt.Printf("Duração média: %.0f ms\n", *total.AvgDurationMS)
	}
	if total.AvgTTFTMS != nil {
		fmt.Printf("TTFT médio:    %.0f ms\n", *total.AvgTTFTMS)
	}
	if total.AvgOutputTokensPerSecond != nil {
		fmt.Printf("Throughput:    %.1f tok/s\n", *total.AvgOutputTokensPerSecond)
	}
	fmt.Printf("Custo reportado (provedor): %d µUSD\n", total.ProviderReportedMicroUSD)
	fmt.Printf("Equivalente API: %d µUSD; inferido: %d µUSD; não classificado: %d µUSD\n",
		total.APIEquivalentMicroUSD, total.InferredMicroUSD, total.UnclassifiedMicroUSD)
	if len(report.ByModel) > 0 {
		fmt.Println("Top modelos:")
		limit := len(report.ByModel)
		if limit > 5 {
			limit = 5
		}
		for _, row := range report.ByModel[:limit] {
			model := strings.TrimSpace(row.Provider + "/" + row.Model)
			failure := ""
			if row.ErrorRate != nil {
				failure = fmt.Sprintf(", falhas: %d (%.1f%%)", row.InvocationsFailed, *row.ErrorRate*100)
			}
			fmt.Printf("  - %-40s obs: %d%s\n", model, row.ObservationCount, failure)
		}
	}
	return nil
}
