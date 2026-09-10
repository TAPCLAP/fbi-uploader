package facebook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// WritePrettyJSON writes body to w. Valid JSON is indented; otherwise it is written as-is.
// Empty body is skipped.
func WritePrettyJSON(w io.Writer, body []byte) error {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return nil
	}
	var buf bytes.Buffer
	if err := json.Indent(&buf, trimmed, "", "  "); err != nil {
		_, err = fmt.Fprintf(w, "%s\n", trimmed)
		return err
	}
	buf.WriteByte('\n')
	_, err := w.Write(buf.Bytes())
	return err
}
