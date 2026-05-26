package output

import (
	"fmt"
	"io"
	"strings"

	reportpkg "github.com/openshift-online/finops-tools/cli/internal/report"
	"github.com/olekukonko/tablewriter"
)

// WriteReportTemplateList prints available report templates.
func WriteReportTemplateList(w io.Writer, templates []reportpkg.TemplateInfo) error {
	return writeReportTemplateListPretty(w, templates)
}

func writeReportTemplateListPretty(w io.Writer, templates []reportpkg.TemplateInfo) error {
	s := newStyler(w)
	if len(templates) == 0 {
		msg := "No report templates available."
		if s.enabled {
			msg = s.dim(msg)
		}
		_, err := fmt.Fprintln(w, msg)
		return err
	}

	title := "Available report templates:"
	if s.enabled {
		title = s.bold(s.cyan(title))
	}
	if _, err := fmt.Fprintln(w, title); err != nil {
		return err
	}

	table := tablewriter.NewWriter(w)
	table.SetAutoWrapText(false)
	table.SetBorder(false)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetTablePadding("\t")
	table.SetHeader([]string{
		cell(s, s.bold, "TEMPLATE"),
		cell(s, s.bold, "FORMATS"),
		cell(s, s.bold, "DESCRIPTION"),
	})

	for _, t := range templates {
		formats := strings.Join(t.Formats, ", ")
		if s.enabled {
			formats = s.dim(formats)
		}
		table.Append([]string{
			cell(s, s.bold, t.Name),
			formats,
			t.Description,
		})
	}
	table.Render()
	return nil
}
