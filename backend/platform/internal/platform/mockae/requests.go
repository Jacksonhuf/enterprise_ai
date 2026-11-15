package mockae

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	api "github.com/Tencent/WeKnora/internal/platform/api"
	"io"
	"net/http"
	"strings"
)

// Digest the received JSON shape, preserving omitted optional fields. Both execution
// paths use sorted object keys, so callers can recover with the same canonical request.
func decodeRequest(r *http.Request, value any) (string, error) {
	raw, e := io.ReadAll(io.LimitReader(r.Body, (1<<20)+1))
	if e != nil || len(raw) > 1<<20 {
		return "", &api.Error{Status: 400, Code: "INVALID_REQUEST", Message: "请求大小或内容无效"}
	}
	r.Body = io.NopCloser(bytes.NewReader(raw))
	if e = api.Decode(r, value); e != nil {
		return "", e
	}
	var object any
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	if e = d.Decode(&object); e != nil {
		return "", e
	}
	canonical, _ := json.Marshal(object)
	sum := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func validDigest(value string) bool {
	if len(value) != 71 || !strings.HasPrefix(value, "sha256:") || strings.ToLower(value) != value {
		return false
	}
	_, e := hex.DecodeString(value[7:])
	return e == nil
}
