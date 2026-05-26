package report

import "testing"

func TestTemplatesIncludesCosts(t *testing.T) {
	templates := Templates()
	if len(templates) == 0 {
		t.Fatal("expected at least one template")
	}
	found := false
	for _, t := range templates {
		if t.Name == TemplateCosts {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("templates = %+v", templates)
	}
}
