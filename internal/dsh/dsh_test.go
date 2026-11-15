package dsh

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func writeBridge(t *testing.T, port int, token string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "dsh-bridge.json")
	data := []byte(`{"version":1,"port":` + strconv.Itoa(port) + `,"token":"` + token + `","pid":1}`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("QG_DSH_BRIDGE", path)
}

func TestOpenWorkspaceUsesBridge(t *testing.T) {
	dir := t.TempDir()
	var gotPath, gotAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		var body struct {
			Path string `json:"path"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		gotPath = body.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"workspaceId":"ws-1","created":true}`))
	}))
	defer server.Close()

	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil {
		t.Fatal(err)
	}
	writeBridge(t, port, "secret")

	res, err := OpenWorkspace(dir)
	if err != nil {
		t.Fatalf("OpenWorkspace() error = %v", err)
	}
	if gotAuth != "Bearer secret" {
		t.Errorf("Authorization = %q, want Bearer secret", gotAuth)
	}
	if gotPath != dir {
		t.Errorf("path = %q, want %q", gotPath, dir)
	}
	if res.WorkspaceID != "ws-1" || !res.Created {
		t.Errorf("result = %+v, want workspace ws-1 created", res)
	}
}

func TestReadBridgeRejectsUnsupportedVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dsh-bridge.json")
	if err := os.WriteFile(path, []byte(`{"version":2,"port":1234,"token":"x"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("QG_DSH_BRIDGE", path)

	if _, err := readBridge(); err == nil {
		t.Fatal("readBridge() accepted an unsupported version")
	}
}

func TestOpenWorkspaceRejectsMissingDirectory(t *testing.T) {
	t.Setenv("QG_DSH_NO_LAUNCH", "1")

	_, err := OpenWorkspace(filepath.Join(t.TempDir(), "missing"))
	if err == nil {
		t.Fatal("OpenWorkspace() accepted a missing directory")
	}
}

func TestCallOpenRejectsUnauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"ok":false,"error":"unauthorized"}`))
	}))
	defer server.Close()

	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil {
		t.Fatal(err)
	}

	_, err = callOpen(&bridgeInfo{Version: bridgeVersion, Port: port, Token: "wrong"}, t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "令牌") {
		t.Fatalf("callOpen() error = %v, want token error", err)
	}
}

func TestCallOpenRejectsBadRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"ok":false,"error":"path is not a directory"}`))
	}))
	defer server.Close()

	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil {
		t.Fatal(err)
	}

	_, err = callOpen(&bridgeInfo{Version: bridgeVersion, Port: port, Token: "secret"}, t.TempDir())
	var rejected *rejectedError
	if !errors.As(err, &rejected) {
		t.Fatalf("callOpen() error = %v, want rejectedError", err)
	}
}

func TestRemoveWorkspaceUsesBridge(t *testing.T) {
	dir := t.TempDir()
	var gotPath, gotAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		var body struct {
			Path string `json:"path"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		gotPath = body.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"removed":true,"workspaceId":"ws-1"}`))
	}))
	defer server.Close()

	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil {
		t.Fatal(err)
	}
	writeBridge(t, port, "secret")

	res, err := RemoveWorkspace(dir)
	if err != nil {
		t.Fatalf("RemoveWorkspace() error = %v", err)
	}
	if gotAuth != "Bearer secret" {
		t.Errorf("Authorization = %q, want Bearer secret", gotAuth)
	}
	if gotPath != dir {
		t.Errorf("path = %q, want %q", gotPath, dir)
	}
	if !res.Removed || res.WorkspaceID != "ws-1" {
		t.Errorf("result = %+v, want removed workspace ws-1", res)
	}
}

func TestRemoveWorkspaceWithoutBridgeIsNoop(t *testing.T) {
	t.Setenv("QG_DSH_BRIDGE", filepath.Join(t.TempDir(), "missing.json"))

	res, err := RemoveWorkspace(t.TempDir())
	if err != nil {
		t.Fatalf("RemoveWorkspace() error = %v", err)
	}
	if res.Removed {
		t.Errorf("result = %+v, want no-op", res)
	}
}
