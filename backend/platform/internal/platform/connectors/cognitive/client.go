package cognitive

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/Tencent/WeKnora/internal/platform/applications"
	"github.com/Tencent/WeKnora/internal/platform/governance"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	BaseURL      string
	ServiceToken string
	SigningKey   []byte
	HTTP         *http.Client
}
type RemoteError struct {
	Status int
	Body   string
}

func (e *RemoteError) Error() string { return fmt.Sprintf("认知服务返回状态 %d", e.Status) }
func (c *Client) Do(ctx context.Context, p governance.Principal, method, path string, body any, key string) (json.RawMessage, error) {
	var data []byte
	var err error
	if body != nil {
		data, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(c.BaseURL, "/")+path, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.ServiceToken)
	req.Header.Set("X-Delegation-Token", governance.Sign(p, c.SigningKey))
	req.Header.Set("X-Request-ID", applications.ID("request"))
	req.Header.Set("X-Invocation-Purpose", p.Purpose)
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	h := c.HTTP
	if h == nil {
		h = &http.Client{Timeout: 15 * time.Second}
	}
	resp, err := h.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20+1))
	if err != nil {
		return nil, err
	}
	if len(b) > 1<<20 {
		return nil, fmt.Errorf("认知服务响应超过限制")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &RemoteError{resp.StatusCode, string(b)}
	}
	if !json.Valid(b) {
		return nil, fmt.Errorf("认知服务响应格式错误")
	}
	return b, nil
}
