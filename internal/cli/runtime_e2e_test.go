//go:build e2e

package cli

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Miss-you/go-symphony/internal/domain"
	"github.com/Miss-you/go-symphony/internal/trackers/memory"
)

func TestRuntimeE2EServesDashboardAPIAndShutsDown(t *testing.T) {
	root := t.TempDir()
	store := newTestStoreWithServer(t, root, "memory", "", 34567)
	reader := memory.NewReader([]domain.WorkItem{testWorkItem("item-e2e", "MT-E2E", "In Progress")})
	transports := newRuntimeTransportFactory(newRuntimeTransport(
		`{"id":1,"result":{}}`,
		`{"id":2,"result":{"thread":{"id":"thread-e2e"}}}`,
		`{"id":3,"result":{"turn":{"id":"turn-1"}}}`,
		`{"method":"turn/completed","params":{"usage":{"input_tokens":1,"output_tokens":2,"total_tokens":3}}}`,
	))
	override := 0

	rt, err := StartRuntime(context.Background(), RuntimeOptions{
		Store:              store,
		Reader:             reader,
		TransportFactory:   transports.Start,
		ServerPortOverride: &override,
	})
	if err != nil {
		t.Fatalf("StartRuntime returned error: %v", err)
	}

	baseURL := strings.TrimRight(rt.DashboardURL(), "/")
	if baseURL == "" {
		t.Fatal("DashboardURL is empty, want e2e listener")
	}
	if body := httpGet(t, baseURL+"/"); !strings.Contains(body, "Operations Dashboard") {
		t.Fatalf("dashboard body missing heading:\n%s", body)
	}
	if body := httpGet(t, baseURL+"/api/v1/state"); !strings.Contains(body, `"generated_at"`) {
		t.Fatalf("state body = %s, want API payload", body)
	}
	waitFor(t, time.Second, func() bool {
		return len(transports.usedTransports()) == 1 && findWriteByMethodOrNil(transports.transport(0).writes(), "turn/start") != nil
	})

	if err := rt.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	client := &http.Client{Timeout: 200 * time.Millisecond}
	resp, err := client.Get(baseURL + "/api/v1/state")
	if err == nil {
		_ = resp.Body.Close()
		t.Fatalf("GET after Close unexpectedly succeeded; listener may still be serving")
	}
}
