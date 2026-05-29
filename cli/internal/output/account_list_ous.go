// account_list_ous.go formats AWS Organizational Units for terminal output.
package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
)

// OrganizationalUnitRow is one OU for list display.
type OrganizationalUnitRow struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// OrganizationalUnitListView is the rendered OU list payload.
type OrganizationalUnitListView struct {
	PayerAlias string                  `json:"payer_alias"`
	ParentID   string                  `json:"parent_id,omitempty"`
	Units      []OrganizationalUnitRow `json:"units"`
}

// WriteOrganizationalUnitListResult renders OUs in the selected format.
func WriteOrganizationalUnitListResult(w io.Writer, format Format, view OrganizationalUnitListView) error {
	switch format {
	case FormatPrettyPrint:
		return writeOrganizationalUnitListPretty(w, view)
	case FormatJSON:
		return writeOrganizationalUnitListJSON(w, view)
	case FormatCSV:
		return writeOrganizationalUnitListCSV(w, view)
	default:
		return fmt.Errorf("unknown format %q", format)
	}
}

func writeOrganizationalUnitListJSON(w io.Writer, view OrganizationalUnitListView) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(view)
}

func writeOrganizationalUnitListCSV(w io.Writer, view OrganizationalUnitListView) error {
	cw := csv.NewWriter(w)
	defer cw.Flush()

	if err := cw.Write([]string{"payer_alias", "parent_id", "ou_id", "ou_name"}); err != nil {
		return err
	}
	for _, ou := range view.Units {
		if err := cw.Write([]string{view.PayerAlias, view.ParentID, ou.ID, ou.Name}); err != nil {
			return err
		}
	}
	return cw.Error()
}
