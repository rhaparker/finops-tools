package report

import (
	"testing"

	"github.com/openshift-online/finops-tools/core/cost"
)

func TestGeneratorForKnownTemplates(t *testing.T) {
	for _, name := range []string{TemplateCosts, TemplateSavingsPlans, TemplateCostAnomalies} {
		if _, err := GeneratorFor(name); err != nil {
			t.Fatalf("GeneratorFor(%q): %v", name, err)
		}
	}
}

func TestCostAnomaliesGeneratorRequiresTargets(t *testing.T) {
	gen, err := GeneratorFor(TemplateCostAnomalies)
	if err != nil {
		t.Fatal(err)
	}
	err = gen.Validate(GenerateInput{Format: FormatHTML})
	if err == nil {
		t.Fatal("expected error for zero targets")
	}
}

func TestSavingsPlansGeneratorRequiresTargets(t *testing.T) {
	gen, err := GeneratorFor(TemplateSavingsPlans)
	if err != nil {
		t.Fatal(err)
	}
	err = gen.Validate(GenerateInput{Format: FormatHTML})
	if err == nil {
		t.Fatal("expected error for zero targets")
	}
}

func TestCostsGeneratorAllowsZeroTargets(t *testing.T) {
	gen, err := GeneratorFor(TemplateCosts)
	if err != nil {
		t.Fatal(err)
	}
	if err := gen.Validate(GenerateInput{Format: FormatHTML}); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func TestGeneratorValidateRejectsUnsupportedFormat(t *testing.T) {
	gen, err := GeneratorFor(TemplateCosts)
	if err != nil {
		t.Fatal(err)
	}
	err = gen.Validate(GenerateInput{
		Format:  "pdf",
		Targets: []cost.AccountTarget{{AccountID: "111111111111"}},
	})
	if err == nil {
		t.Fatal("expected format error")
	}
}
