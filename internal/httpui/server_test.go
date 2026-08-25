package httpui

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"seed-vault-release/internal/application"
	"seed-vault-release/internal/repository"
)

func testServer(t *testing.T) *httptest.Server {
	t.Helper()
	store, err := repository.Open(filepath.Join(t.TempDir(), "http.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	server := httptest.NewServer(New(application.New(store)).Handler())
	t.Cleanup(server.Close)
	return server
}

func TestPageHealthAndCreateCase(t *testing.T) {
	server := testServer(t)
	for _, path := range []string{"/", "/assets/app.css", "/assets/workflow.css", "/assets/app.js", "/healthz"} {
		res, err := http.Get(server.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != http.StatusOK {
			t.Fatalf("GET %s=%d", path, res.StatusCode)
		}
	}
	now := time.Now().UTC()
	body := map[string]any{"actor": "接收员甲", "role": "receiver", "idempotencyKey": "http-create", "accessionCode": "HTTP-1", "speciesName": "珙桐", "collectionSite": "四川卧龙", "collectedAt": now.Add(-time.Hour), "receivedAt": now, "owner": "接收员甲"}
	b, _ := json.Marshal(body)
	res, err := http.Post(server.URL+"/api/v1/cases", "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("创建案卷返回%d", res.StatusCode)
	}
	if res.Header.Get("Location") == "" {
		t.Fatal("创建响应缺少Location")
	}
}

func TestRejectsUnknownJSONField(t *testing.T) {
	server := testServer(t)
	res, err := http.Post(server.URL+"/api/v1/cases", "application/json", bytes.NewBufferString(`{"unknown":true}`))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("未知字段应返回400，实际%d", res.StatusCode)
	}
}
