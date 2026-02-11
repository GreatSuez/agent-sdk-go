package eval

import (
	"fmt"
	"sort"
	"strings"
)

func FormatMarkdown(report Report) string {
	var b strings.Builder
	b.WriteString("# Eval Report\n\n")
	b.WriteString(fmt.Sprintf("- dataset: `%s`\n", report.Dataset))
	b.WriteString(fmt.Sprintf("- provider: `%s`\n", report.Provider))
	b.WriteString(fmt.Sprintf("- pass rate: `%.2f%%` (%d/%d)\n", report.PassRate, report.Passed, report.Total))
	b.WriteString(fmt.Sprintf("- latency: avg `%.2fms`, p50 `%dms`, p95 `%dms`\n", report.AvgLatencyMs, report.LatencyP50Ms, report.LatencyP95Ms))
	b.WriteString(fmt.Sprintf("- tokens: in `%d`, out `%d`, total `%d`\n", report.TotalInputTokens, report.TotalOutputTokens, report.TotalTokens))
	b.WriteString(fmt.Sprintf("- tool constraint accuracy: `%.2f%%` (%d/%d)\n", report.ToolConstraintAccuracy, report.ToolConstraintPassed, report.ToolConstraintCases))

	if len(report.PerTag) > 0 {
		keys := make([]string, 0, len(report.PerTag))
		for k := range report.PerTag {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		b.WriteString("\n## Per Tag\n\n")
		for _, tag := range keys {
			m := report.PerTag[tag]
			b.WriteString(fmt.Sprintf("- `%s`: `%.2f%%` (%d/%d)\n", tag, m.PassRate, m.Passed, m.Total))
		}
	}

	failing := make([]CaseResult, 0)
	for _, c := range report.Results {
		if !c.Pass {
			failing = append(failing, c)
		}
	}

	if len(failing) > 0 {
		b.WriteString("\n## Failures\n\n")
		for _, c := range failing {
			reason := c.Error
			if strings.TrimSpace(reason) == "" {
				reason = firstFailedCheck(c.Checks)
			}
			if strings.TrimSpace(reason) == "" {
				reason = "unknown failure"
			}
			b.WriteString(fmt.Sprintf("- `%s`: %s\n", c.CaseID, reason))
		}
	}

	return b.String()
}

func firstFailedCheck(checks []CheckResult) string {
	for _, check := range checks {
		if !check.Pass {
			if strings.TrimSpace(check.Detail) != "" {
				return fmt.Sprintf("%s (%s)", check.Name, check.Detail)
			}
			return check.Name
		}
	}
	return ""
}
