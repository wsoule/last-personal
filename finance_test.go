package main

import (
	"html/template"
	"math"
	"testing"
	"time"
)

func TestCalculateContributionGrowthZeroReturn(t *testing.T) {
	result := calculateContributionGrowth(ContributionGrowthRequest{
		ContributionAmount: 100,
		StartingAmount:     500,
		AnnualReturn:       0,
		Frequency:          "monthly",
		StartDate:          time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
		MaxPeriods:         12,
	})

	if !almostEqual(result.TotalContributions, 1200) {
		t.Fatalf("TotalContributions = %.2f, want 1200.00", result.TotalContributions)
	}
	if !almostEqual(result.ProjectedValue, 1700) {
		t.Fatalf("ProjectedValue = %.2f, want 1700.00", result.ProjectedValue)
	}
	if !almostEqual(result.InterestEarned, 0) {
		t.Fatalf("InterestEarned = %.2f, want 0.00", result.InterestEarned)
	}
	if len(result.Rows) != 13 {
		t.Fatalf("len(Rows) = %d, want 13", len(result.Rows))
	}
}

func TestCalculateContributionGrowthTargetHighlight(t *testing.T) {
	result := calculateContributionGrowth(ContributionGrowthRequest{
		ContributionAmount: 100,
		StartingAmount:     500,
		AnnualReturn:       0,
		TargetAmount:       1000,
		Frequency:          "monthly",
		StartDate:          time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
		MaxPeriods:         12,
	})

	if !result.TargetReached {
		t.Fatal("TargetReached = false, want true")
	}
	if result.TargetDate != "Jun 01, 2026" {
		t.Fatalf("TargetDate = %q, want Jun 01, 2026", result.TargetDate)
	}
	if len(result.Rows) != 6 {
		t.Fatalf("len(Rows) = %d, want 6", len(result.Rows))
	}
	if !result.Rows[len(result.Rows)-1].TargetReached {
		t.Fatal("final row should be highlighted as target reached")
	}
}

func TestTemplatesParse(t *testing.T) {
	funcMap := template.FuncMap{
		"currentYear": func() int { return 2026 },
	}

	parsedTemplates := template.Must(template.New("").Funcs(funcMap).ParseGlob("templates/*.html"))
	template.Must(parsedTemplates.ParseGlob("templates/partials/*.html"))
	template.Must(parsedTemplates.ParseGlob("templates/finance/*.html"))
}

func almostEqual(got, want float64) bool {
	return math.Abs(got-want) < 0.0001
}
