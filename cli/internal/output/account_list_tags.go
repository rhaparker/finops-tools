// account_list_tags.go formats AWS account tags for terminal output.
package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
)

// AccountTagRow is one account tag for display.
type AccountTagRow struct {
	Key   string
	Value string
}

// AccountTagsView is the rendered AWS account + tags payload.
type AccountTagsView struct {
	AccountID string
	Alias     string
	Tags      []AccountTagRow
}

// WriteAWSAccountTags renders AWS Organizations account tags.
func WriteAWSAccountTags(w io.Writer, view AccountTagsView) error {
	return writeAWSAccountTagsPretty(w, view)
}

// WriteAWSAccountTagsResult renders AWS Organizations account tags in the selected format.
func WriteAWSAccountTagsResult(w io.Writer, format Format, view AccountTagsView) error {
	switch format {
	case FormatPrettyPrint:
		return writeAWSAccountTagsPretty(w, view)
	case FormatJSON:
		return writeAWSAccountTagsJSON(w, view)
	case FormatCSV:
		return writeAWSAccountTagsCSV(w, view)
	default:
		return fmt.Errorf("unknown format %q", format)
	}
}

func writeAWSAccountTagsJSON(w io.Writer, view AccountTagsView) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(view)
}

func writeAWSAccountTagsCSV(w io.Writer, view AccountTagsView) error {
	cw := csv.NewWriter(w)
	defer cw.Flush()

	if err := cw.Write([]string{"account_id", "alias", "key", "value"}); err != nil {
		return err
	}
	for _, tag := range view.Tags {
		if err := cw.Write([]string{view.AccountID, view.Alias, tag.Key, tag.Value}); err != nil {
			return err
		}
	}
	return cw.Error()
}
