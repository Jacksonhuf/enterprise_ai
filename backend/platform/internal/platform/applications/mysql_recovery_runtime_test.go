package applications

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/Tencent/WeKnora/internal/platform/testsupport"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"testing"
	"time"
)

func recoveryPort(t *testing.T) int {
	t.Helper()
	listener, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	return port
}
func recoveryHost(t *testing.T, binary, dsn string) (string, func()) {
	t.Helper()
	port, mock := recoveryPort(t), recoveryPort(t)
	for mock == port {
		mock = recoveryPort(t)
	}
	cmd := exec.Command(binary, "--demo", "--storage", "mysql", "--mysql-dsn-env", "COGNARA_RECOVERY_DSN", "--port", strconv.Itoa(port), "--mock-port", strconv.Itoa(mock))
	cmd.Env = append(os.Environ(), "COGNARA_RECOVERY_DSN="+dsn)
	if e := cmd.Start(); e != nil {
		t.Fatal("compiled recovery host failed to start")
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait(); close(done) }()
	var once sync.Once
	stop := func() {
		once.Do(func() {
			_ = cmd.Process.Signal(syscall.SIGTERM)
			select {
			case e := <-done:
				if e != nil {
					t.Error("recovery host exited unsuccessfully")
				}
			case <-time.After(10 * time.Second):
				_ = cmd.Process.Kill()
				<-done
				t.Error("recovery host did not stop gracefully")
			}
		})
	}
	t.Cleanup(stop)
	base := fmt.Sprintf("http://127.0.0.1:%d", port)
	client := http.Client{Timeout: time.Second}
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		response, e := client.Get(base + "/ready")
		if e == nil {
			io.Copy(io.Discard, response.Body)
			response.Body.Close()
			if response.StatusCode == 200 {
				return base, stop
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("recovery host did not become ready")
	return "", stop
}
func recoveryRequest(t *testing.T, base, actor, method, path string, body any, want int) []byte {
	t.Helper()
	var input io.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		input = bytes.NewReader(raw)
	}
	request, e := http.NewRequest(method, base+path, input)
	if e != nil {
		t.Fatal(e)
	}
	request.Header.Set("Authorization", "Bearer "+actor)
	request.Header.Set("Content-Type", "application/json")
	client := http.Client{Timeout: 20 * time.Second}
	response, e := client.Do(request)
	if e != nil {
		t.Fatal("recovery HTTP request failed")
	}
	defer response.Body.Close()
	raw, e := io.ReadAll(response.Body)
	if e != nil || response.StatusCode != want {
		t.Fatalf("recovery %s %s: got %d want %d", method, path, response.StatusCode, want)
	}
	return raw
}
func recoveryObject(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	var value map[string]any
	if e := json.Unmarshal(raw, &value); e != nil {
		t.Fatal("invalid recovery response")
	}
	return value
}
func TestCompiledMySQLV2RecoveryPreservesRuntimePermissionsAndUsage(t *testing.T) {
	runtime, exporter, importer := os.Getenv("COGNARA_TEST_RUNTIME_BINARY"), os.Getenv("COGNARA_TEST_EXPORT_BINARY"), os.Getenv("COGNARA_TEST_IMPORT_BINARY")
	if runtime == "" || exporter == "" || importer == "" {
		if os.Getenv("COGNARA_REQUIRE_MYSQL_TESTS") == "1" {
			t.Fatal("compiled recovery binaries required")
		}
		t.Skip("compiled recovery binaries not configured")
	}
	source, target := testsupport.MySQLDSN(t), testsupport.MySQLDSN(t)
	s, e := OpenMySQL(source)
	if e != nil {
		t.Fatal(e)
	}
	s.Close()
	if e = MigrateMySQLAudit(context.Background(), source); e != nil {
		t.Fatal(e)
	}
	base, stop := recoveryHost(t, runtime, source)
	session := recoveryObject(t, recoveryRequest(t, base, "demo-user", "POST", "/api/v1/cognitive/apps/channel-replenishment/sessions", map[string]any{}, 200))["session_id"].(string)
	stream := recoveryRequest(t, base, "demo-user", "POST", "/api/v1/agent-chat/"+session, map[string]any{"query": "SO1002需求增加到350"}, 200)
	if !bytes.Contains(stream, []byte("276.6")) {
		t.Fatal("source business result missing")
	}
	historyPath := "/api/v1/messages/" + session + "/load"
	history := recoveryObject(t, recoveryRequest(t, base, "demo-user", "GET", historyPath, nil, 200))["data"].([]any)
	if len(history) != 2 {
		t.Fatal("source turn not persisted")
	}
	answer := history[1].(map[string]any)
	usagePath := "/api/v1/cognitive/governance/usage"
	before := recoveryObject(t, recoveryRequest(t, base, "demo-admin", "GET", usagePath, nil, 200))["items"]
	stop()
	snapshot := filepath.Join(t.TempDir(), "runtime-v2.json")
	export := exec.Command(exporter, "--mysql-dsn-env", "COGNARA_RECOVERY_DSN", "--output", snapshot)
	export.Env = append(os.Environ(), "COGNARA_RECOVERY_DSN="+source)
	raw, e := export.CombinedOutput()
	if e != nil {
		t.Fatal("compiled runtime snapshot export failed")
	}
	var report SnapshotReport
	if json.Unmarshal(raw, &report) != nil || report.SourceSchemaVersion != 2 || report.Messages != 2 {
		t.Fatal("runtime snapshot summary invalid")
	}
	restore := exec.Command(importer, "--mysql-dsn-env", "COGNARA_RECOVERY_DSN", "--input", snapshot, "--expected-sha256", report.SHA256, "--source-schema-version", "2")
	restore.Env = append(os.Environ(), "COGNARA_RECOVERY_DSN="+target)
	if _, e = restore.CombinedOutput(); e != nil {
		t.Fatal("compiled runtime restore failed")
	}
	recovered, stopRecovered := recoveryHost(t, runtime, target)
	historyRestored := recoveryRequest(t, recovered, "demo-user", "GET", historyPath, nil, 200)
	if !bytes.Contains(historyRestored, []byte("276.6")) {
		t.Fatal("recovered history lost result")
	}
	recoveryRequest(t, recovered, "demo-outsider", "GET", historyPath, nil, 404)
	replay := "/api/v1/sessions/continue-stream/" + session + "?message_id=" + answer["id"].(string)
	restoredStream := recoveryRequest(t, recovered, "demo-user", "GET", replay, nil, 200)
	if !bytes.Contains(restoredStream, []byte("276.6")) {
		t.Fatal("recovered replay failed")
	}
	after := recoveryObject(t, recoveryRequest(t, recovered, "demo-admin", "GET", usagePath, nil, 200))["items"]
	beforeJSON, _ := json.Marshal(before)
	afterJSON, _ := json.Marshal(after)
	if !bytes.Equal(beforeJSON, afterJSON) {
		t.Fatal("restore/replay repeated usage")
	}

	fresh := recoveryRequest(t, recovered, "demo-user", "POST", "/api/v1/agent-chat/"+session, map[string]any{"query": "SO1002需求增加到350"}, 200)
	if !bytes.Contains(fresh, []byte("276.6")) {
		t.Fatal("restored host cannot execute a new turn")
	}
	freshUsage := recoveryObject(t, recoveryRequest(t, recovered, "demo-admin", "GET", usagePath, nil, 200))["items"].([]any)
	if len(freshUsage) != len(after.([]any))+1 {
		t.Fatal("new turn usage was not counted exactly once")
	}
	recoveryRequest(t, recovered, "demo-admin", "DELETE", "/api/v1/cognitive/governance/grants/grant-service", nil, 200)
	redacted := recoveryRequest(t, recovered, "demo-user", "GET", historyPath, nil, 200)
	if bytes.Contains(redacted, []byte("276.6")) || !bytes.Contains(redacted, []byte(`"content_redacted":true`)) {
		t.Fatal("restored permissions not enforced")
	}
	recoveryRequest(t, recovered, "demo-user", "GET", replay, nil, 403)
	state, e := OpenMySQL(target)
	if e != nil {
		t.Fatal(e)
	}
	defer state.Close()
	if len(state.Read().Audit) <= report.AuditEvents {
		t.Fatal("restored audit did not accept new events")
	}
	stopRecovered()
}
