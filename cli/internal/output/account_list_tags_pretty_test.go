package output

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestWriteAWSAccountTagsPretty(t *testing.T) {
	view := AccountTagsView{
		AccountID: "111111111111",
		Alias:     "osd-tenant-1",
		Tags: []AccountTagRow{
			{Key: "env", Value: "prod"},
			{Key: "owner", Value: "team-a"},
		},
	}

	var buf strings.Builder
	if err := WriteAWSAccountTags(&buf, view); err != nil {
		t.Fatal(err)
	}
	out := stripANSI(buf.String())
	for _, want := range []string{
		"Account:",
		"osd-tenant-1 (111111111111)",
		"AWS account tags",
		"KEY",
		"VALUE",
		"env",
		"prod",
		"owner",
		"team-a",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestWriteAWSAccountTagsPrettyEmpty(t *testing.T) {
	view := AccountTagsView{AccountID: "123456789012"}
	var buf strings.Builder
	if err := WriteAWSAccountTags(&buf, view); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "No AWS account tags found for 123456789012.") {
		t.Fatalf("output = %q", buf.String())
	}
}

func TestWriteAWSAccountTagsResultJSON(t *testing.T) {
	view := AccountTagsView{
		AccountID: "123456789012",
		Alias:     "rh-control",
		Tags: []AccountTagRow{
			{Key: "env", Value: "prod"},
		},
	}

	var buf strings.Builder
	if err := WriteAWSAccountTagsResult(&buf, FormatJSON, view); err != nil {
		t.Fatal(err)
	}

	var got AccountTagsView
	if err := json.Unmarshal([]byte(buf.String()), &got); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if got.AccountID != view.AccountID || len(got.Tags) != 1 || got.Tags[0].Key != "env" {
		t.Fatalf("decoded view = %+v", got)
	}
}

func TestWriteAWSAccountTagsResultCSV(t *testing.T) {
	view := AccountTagsView{
		AccountID: "123456789012",
		Alias:     "rh-control",
		Tags: []AccountTagRow{
			{Key: "env", Value: "prod"},
		},
	}

	var buf strings.Builder
	if err := WriteAWSAccountTagsResult(&buf, FormatCSV, view); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{
		"account_id,alias,key,value",
		"123456789012,rh-control,env,prod",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}
