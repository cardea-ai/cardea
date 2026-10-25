package policy

import (
	"context"
	"testing"

	"github.com/cardea-ai/cardea/internal/plugin"
)

func TestNFR_AV_02_DefaultDenyWhenFindingsPresentAndGradeTooHigh(t *testing.T) {
	eng, err := New(context.Background())
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	decision, err := eng.Evaluate(context.Background(), plugin.PolicyInput{
		Grade:    plugin.GradeTopSecret,
		Findings: []plugin.Finding{{EntityType: "KR_RRN"}},
	})
	if err != nil {
		t.Fatalf("Evaluate error: %v", err)
	}
	if decision != plugin.DecisionDeny {
		t.Errorf("Evaluate() = %v, want deny for top-secret grade", decision)
	}
}

func TestNFR_AV_02_AllowWhenNoFindingsAndLowGrade(t *testing.T) {
	eng, err := New(context.Background())
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	decision, err := eng.Evaluate(context.Background(), plugin.PolicyInput{
		Grade: plugin.GradePublic,
	})
	if err != nil {
		t.Fatalf("Evaluate error: %v", err)
	}
	if decision != plugin.DecisionAllow {
		t.Errorf("Evaluate() = %v, want allow for public grade with no findings", decision)
	}
}

func TestNFR_AV_02_PartialWhenFindingsPresentButGradeModerate(t *testing.T) {
	eng, err := New(context.Background())
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	decision, err := eng.Evaluate(context.Background(), plugin.PolicyInput{
		Grade:    plugin.GradeConfidential,
		Findings: []plugin.Finding{{EntityType: "KR_RRN"}},
	})
	if err != nil {
		t.Fatalf("Evaluate error: %v", err)
	}
	if decision != plugin.DecisionPartial {
		t.Errorf("Evaluate() = %v, want partial", decision)
	}
}
