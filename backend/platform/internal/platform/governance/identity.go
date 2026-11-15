package governance

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type Principal struct {
	TenantID      string `json:"tenant_id"`
	UserID        string `json:"actor_user_id"`
	Name          string `json:"name"`
	Role          string `json:"role"`
	Scope         string `json:"scope"`
	ApplicationID string `json:"application_id,omitempty"`
	ReleaseID     string `json:"application_release_id,omitempty"`
	TurnID        string `json:"turn_id,omitempty"`
	Audience      string `json:"audience"`
	Purpose       string `json:"purpose"`
	Expires       int64  `json:"expires"`
}

func DemoIdentity(token string) (Principal, error) {
	p := Principal{TenantID: "42", Scope: "east", Audience: "cognitive-business"}
	switch token {
	case "demo-user":
		p.UserID = token
		p.Name = "张敏"
		p.Role = "user"
	case "demo-engineer":
		p.UserID = token
		p.Name = "应用开发者"
		p.Role = "engineer"
	case "demo-admin":
		p.UserID = token
		p.Name = "企业与平台管理员"
		p.Role = "admin"
	case "demo-outsider":
		p.UserID = token
		p.Name = "隔离验证账户"
		p.Role = "user"
		p.TenantID = "43"
		p.Scope = "west"
	default:
		return p, errors.New("invalid demo credential")
	}
	return p, nil
}
func (p Principal) CanBuild() bool { return p.Role == "engineer" || p.Role == "admin" }
func Sign(p Principal, key []byte) string {
	p.Expires = time.Now().Add(5 * time.Minute).Unix()
	b, _ := json.Marshal(p)
	v := base64.RawURLEncoding.EncodeToString(b)
	m := hmac.New(sha256.New, key)
	m.Write([]byte(v))
	return v + "." + base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}
func Verify(token string, key []byte) (Principal, error) {
	var p Principal
	a := strings.Split(token, ".")
	if len(a) != 2 {
		return p, errors.New("invalid delegation")
	}
	sig, e := base64.RawURLEncoding.DecodeString(a[1])
	if e != nil {
		return p, e
	}
	m := hmac.New(sha256.New, key)
	m.Write([]byte(a[0]))
	if !hmac.Equal(sig, m.Sum(nil)) {
		return p, errors.New("invalid delegation")
	}
	b, e := base64.RawURLEncoding.DecodeString(a[0])
	if e != nil {
		return p, e
	}
	if e = json.Unmarshal(b, &p); e != nil {
		return p, e
	}
	if p.Expires <= time.Now().Unix() || p.Audience != "cognitive-business" {
		return p, errors.New("expired or wrong audience")
	}
	return p, nil
}
