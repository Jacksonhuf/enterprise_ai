package runtime

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

func strictDecode(raw []byte, value any) error {
	if len(raw) > 65536 {
		return fmt.Errorf("tool input exceeds limit")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if e := d.Decode(value); e != nil {
		return e
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return fmt.Errorf("invalid trailing JSON")
	}
	return nil
}
