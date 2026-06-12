package report

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/openshift-online/finops-tools/core/cost"
)

// Stepper reports long-running steps while generating a report.
type Stepper interface {
	Step(message string)
}

// GenerateInput is shared context for building and rendering a report template.
type GenerateInput struct {
	Format   string
	Out      io.Writer
	Targets  []cost.AccountTarget
	Range    cost.DateRange
	Progress Stepper
	Now      time.Time
}

// Generator builds and renders one report template.
type Generator interface {
	Validate(in GenerateInput) error
	Generate(ctx context.Context, in GenerateInput) error
}

// GeneratorFor returns the generator registered for a parsed template name.
func GeneratorFor(name string) (Generator, error) {
	g, ok := generators[name]
	if !ok {
		return nil, fmt.Errorf("unsupported template %q", name)
	}
	return g, nil
}

var generators = map[string]Generator{
	TemplateCosts:         costsGenerator{},
	TemplateSavingsPlans:  savingsPlansGenerator{},
	TemplateCostAnomalies: costAnomaliesGenerator{},
}

func validateTemplateFormat(templateName, format string) error {
	for _, t := range Templates() {
		if t.Name != templateName {
			continue
		}
		for _, f := range t.Formats {
			if f == format {
				return nil
			}
		}
		return fmt.Errorf("template %q does not support format %q", templateName, format)
	}
	return fmt.Errorf("unsupported template %q", templateName)
}
