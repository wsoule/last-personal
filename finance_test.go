package main

import (
	"html/template"
	"math"
	"testing"
	"time"
)

func TestCalculateDateSavingsGoalZeroReturn(t *testing.T) {
	result := calculateDateSavingsGoal(DateSavingsGoalRequest{
		GoalAmount:     1200,
		TargetDate:     time.Date(2027, time.January, 1, 0, 0, 0, 0, time.UTC),
		CurrentSavings: 0,
		AnnualReturn:   0,
		Frequency:      "monthly",
		Periods:        12,
		Years:          1,
	})

	if !almostEqual(result.PeriodicContribution, 100) {
		t.Fatalf("PeriodicContribution = %.2f, want 100.00", result.PeriodicContribution)
	}
	if !almostEqual(result.ProjectedValue, 1200) {
		t.Fatalf("ProjectedValue = %.2f, want 1200.00", result.ProjectedValue)
	}
}

func TestCalculateContributionGrowthZeroReturn(t *testing.T) {
	result := calculateContributionGrowth(ContributionGrowthRequest{
		ContributionAmount: 100,
		TargetDate:         time.Date(2027, time.January, 1, 0, 0, 0, 0, time.UTC),
		StartingAmount:     500,
		AnnualReturn:       0,
		Frequency:          "monthly",
		Periods:            12,
		Years:              1,
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
