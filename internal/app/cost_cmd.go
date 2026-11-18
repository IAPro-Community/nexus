package app

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/IAPro-Community/nexus/internal/nexus"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

const (
	costSummaryUsage   = "Usage: nexus cost summary [--days N] [--project-id <id>] [--json]"
	costAnalyticsUsage = "Usage: nexus cost analytics [--days N] [--project-id <id>] [--json]"
)

// costCmd is the entry point of `nexus cost`. It is wiring only: every
// formula comes from the CostApplicationService / canonical store aggregates,
// never duplicated here. All subcommands are read-only over the local cost
// observation ledger.
func costCmd(args []string) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		fmt.Println(costSummaryUsage)
		fmt.Println("  Resumo local de custo sobre o ledger canônico de observações (somente leitura).")
		fmt.Println(costAnalyticsUsage)
		fmt.Println("  Relatório completo de analytics de custo (mesmo schema do endpoint /api/v1/cost/analytics).")
		return nil
	}
	switch args[0] {
	case "summary":
		return costSummaryCmd(args[1:])
	case "analytics":
		return costAnalyticsCmd(args[1:])
	default:
		return fmt.Errorf("unknown cost subcommand %q; %s", args[0], costSummaryUsage)
	}
}

// costWindowFlags holds the shared read-only flags of `nexus cost` subcommands.
type costWindowFlags struct {
	days      int
	projectID string
	isJSON    bool
}

// parseCostWindowFlags parses the flag set shared by `cost summary` and
// `cost analytics`. Days validation mirrors GET /api/v1/cost/analytics:
// 1..365, default 7, same error message, never silently clamped.
func parseCostWindowFlags(args []string, usage string) (costWindowFlags, error) {
	const (
		defaultDays   = 7
		minWindowDays = 1
		maxWindowDays = 365
	)
	flags := costWindowFlags{days: defaultDays}
	parseDays := func(raw string) error {
		value, err := strconv.Atoi(strings.TrimSpace(raw))
		if err != nil || value < minWindowDays || value > maxWindowDays {
			// Same message and boundaries as GET /api/v1/cost/analytics.
			return fmt.Errorf("days must be between 1 and 365")
		}
		flags.days = value
		return nil
	}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--json":
			flags.isJSON = true
		case arg == "--days":
			i++
			if i >= len(args) {
				return flags, fmt.Errorf("--days requires a value; %s", usage)
			}
			if err := parseDays(args[i]); err != nil {
				return flags, err
			}
		case strings.HasPrefix(arg, "--days="):
			if err := parseDays(strings.TrimPrefix(arg, "--days=")); err != nil {
				return flags, err
			}
		case arg == "--project-id":
			i++
			if i >= len(args) {
				return flags, fmt.Errorf("--project-id requires a value; %s", usage)
			}
			flags.projectID = strings.TrimSpace(args[i])
		case strings.HasPrefix(arg, "--project-id="):
			flags.projectID = strings.TrimSpace(strings.TrimPrefix(arg, "--project-id="))
		default:
			return flags, fmt.Errorf("unknown flag %q; %s", arg, usage)
		}
	}
	return flags, nil
}

// fetchCostAnalytics is the single call into CostApplicationService.Analytics —
// the exact same source as GET /api/v1/cost/analytics. No aggregation formula,
// truncation rule or cost-evidence classification is duplicated in the CLI.
func fetchCostAnalytics(projectID string, days int) (nexus.CostAnalyticsReport, error) {
	svc := nexus.NewCostApplicationService(nexus.Default().Store())
	return svc.Analytics(context.Background(), projectID, days)
}

// printCostAnalyticsJSON emits the CostAnalyticsReport integral, with the same
// schema as GET /api/v1/cost/analytics, so CLI, web and TUI share one fixture.
func printCostAnalyticsJSON(report nexus.CostAnalyticsReport) error {
	b, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}

// costSummaryCmd renders `nexus cost summary`. With --json it emits exactly
// the same schema as GET /api/v1/cost/analytics (CostAnalyticsReport), so
// CLI, web and TUI surfaces share one fixture. Days validation mirrors the
// HTTP handler: 1..365, default 7.
func costSummaryCmd(args []string) error {
	flags, err := parseCostWindowFlags(args, costSummaryUsage)
	if err != nil {
		return err
	}
	report, err := fetchCostAnalytics(flags.projectID, flags.days)
	if err != nil {
		return err
	}
	if flags.isJSON {
		return printCostAnalyticsJSON(report)
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

// costAnalyticsCmd renders `nexus cost analytics`. With --json it prints the
// CostAnalyticsReport integral — identical schema to GET /api/v1/cost/analytics
// for the same ledger (source, total with invocation/error observability,
// by_day, by_model, models_truncated). The human path is a summarized view:
// period, observations, cost evidence classes kept separate (provider-reported
// vs API-equivalent vs unknown) and the top 5 models ranked by recorded cost.
func costAnalyticsCmd(args []string) error {
	flags, err := parseCostWindowFlags(args, costAnalyticsUsage)
	if err != nil {
		return err
	}
	report, err := fetchCostAnalytics(flags.projectID, flags.days)
	if err != nil {
		return err
	}
	if flags.isJSON {
		return printCostAnalyticsJSON(report)
	}
	return renderCostAnalyticsHuman(report)
}

// renderCostAnalyticsHuman renders the summarized human view of the analytics
// report. Cost evidence classes stay separate — they are never merged into a
// fictional bill. Unmeasured averages are omitted instead of printed as
// synthetic zeros, and unknown-cost evidence keeps its observation count
// (UNKNOWN != 0): missing classification is never coerced into a fabricated 0.
func renderCostAnalyticsHuman(report nexus.CostAnalyticsReport) error {
	total := report.Total
	fmt.Printf("=== Analytics de custo (%d dias) ===\n", report.Days)
	if report.ProjectID != "" {
		fmt.Printf("Projeto:       %s\n", report.ProjectID)
	}
	fmt.Printf("Fonte:         %s\n", report.Source)
	fmt.Printf("Desde:         %s\n", report.Since.Format("2006-01-02 15:04 UTC"))
	fmt.Printf("Observações:   %d (invocações: %d, falhas: %d)\n",
		total.ObservationCount, total.InvocationsTotal, total.InvocationsFailed)
	if total.ErrorRate != nil {
		fmt.Printf("Taxa de erro:  %.1f%%\n", *total.ErrorRate*100)
	}
	fmt.Printf("Custo reportado (provedor): %d µUSD\n", total.ProviderReportedMicroUSD)
	fmt.Printf("Equivalente API: %d µUSD; inferido: %d µUSD; não classificado: %d µUSD\n",
		total.APIEquivalentMicroUSD, total.InferredMicroUSD, total.UnclassifiedMicroUSD)
	fmt.Printf("Custo desconhecido: %d de %d observações sem custo classificável\n",
		total.UnknownCostObservations, total.ObservationCount)
	if len(report.ByModel) > 0 {
		top := costAnalyticsTopModelsByCost(report.ByModel, 5)
		fmt.Println("Top modelos por custo:")
		for _, row := range top {
			model := strings.TrimSpace(row.Provider + "/" + row.Model)
			failure := ""
			if row.ErrorRate != nil {
				failure = fmt.Sprintf(", falhas: %d (%.1f%%)", row.InvocationsFailed, *row.ErrorRate*100)
			}
			fmt.Printf("  - %-40s %s, obs: %d%s\n", model, costAnalyticsModelCostLabel(row), row.ObservationCount, failure)
		}
	}
	if report.ModelsTruncated {
		fmt.Println("Lista de modelos truncada (mesmo limite do endpoint /api/v1/cost/analytics).")
	}
	return nil
}

// costAnalyticsTopModelsByCost returns up to limit model rows ordered by the
// recorded cost evidence. The ordering is display-only: the classes are still
// rendered separately per row, so no merged balance is ever presented as a
// payable amount. Ties fall back to observation count and stable name order.
func costAnalyticsTopModelsByCost(rows []store.ModelCostAnalytics, limit int) []store.ModelCostAnalytics {
	ordered := make([]store.ModelCostAnalytics, len(rows))
	copy(ordered, rows)
	costEvidence := func(row store.ModelCostAnalytics) int64 {
		return row.ProviderReportedMicroUSD + row.APIEquivalentMicroUSD +
			row.InferredMicroUSD + row.UnclassifiedMicroUSD
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		ci, cj := costEvidence(ordered[i]), costEvidence(ordered[j])
		if ci != cj {
			return ci > cj
		}
		if ordered[i].ObservationCount != ordered[j].ObservationCount {
			return ordered[i].ObservationCount > ordered[j].ObservationCount
		}
		return ordered[i].Provider+ordered[i].Model < ordered[j].Provider+ordered[j].Model
	})
	if len(ordered) > limit {
		ordered = ordered[:limit]
	}
	return ordered
}

// costAnalyticsModelCostLabel describes a model row's cost evidence honestly:
// only classes that actually recorded value are listed; a row without any
// cost evidence reads "custo não medido" instead of a fabricated zero.
func costAnalyticsModelCostLabel(row store.ModelCostAnalytics) string {
	parts := make([]string, 0, 4)
	if row.ProviderReportedMicroUSD > 0 {
		parts = append(parts, fmt.Sprintf("reportado %d µUSD", row.ProviderReportedMicroUSD))
	}
	if row.APIEquivalentMicroUSD > 0 {
		parts = append(parts, fmt.Sprintf("equivalente %d µUSD", row.APIEquivalentMicroUSD))
	}
	if row.InferredMicroUSD > 0 {
		parts = append(parts, fmt.Sprintf("inferido %d µUSD", row.InferredMicroUSD))
	}
	if row.UnclassifiedMicroUSD > 0 {
		parts = append(parts, fmt.Sprintf("não classificado %d µUSD", row.UnclassifiedMicroUSD))
	}
	if len(parts) == 0 {
		return "custo não medido"
	}
	return "custo: " + strings.Join(parts, ", ")
}
