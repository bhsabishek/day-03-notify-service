package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const testKey = "vault-key-WXYZ"

// fakeVault mimics Vault's KV v2 read endpoint.
func fakeVault(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/secret/data/notify" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("X-Vault-Token") != "good-token" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"request_id": "abc",
			"data": {
				"data": {"api_key": "` + testKey + `"},
				"metadata": {"version": 3}
			}
		}`))
	}))
}

func post(t *testing.T, h http.Handler, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("POST", path, bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestVaultKV2KeyIsRead(t *testing.T) {
	v := fakeVault(t)
	defer v.Close()

	key, err := NewSecretStore(v.URL, "good-token").APIKey()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key != testKey {
		t.Fatalf("got %q, want %q", key, testKey)
	}
}

func TestLoadAPIKeyUsesVault(t *testing.T) {
	v := fakeVault(t)
	defer v.Close()

	key, source, err := LoadAPIKey(NewSecretStore(v.URL, "good-token"))
	if err != nil || key != testKey || source != "vault" {
		t.Fatalf("got key=%q source=%q err=%v", key, source, err)
	}
}

func TestLoadAPIKeyFailsWhenVaultUnavailable(t *testing.T) {
	v := fakeVault(t)
	addr := v.URL
	v.Close() // nothing is listening any more

	if _, _, err := LoadAPIKey(NewSecretStore(addr, "good-token")); err == nil {
		t.Fatal("expected an error when Vault is unreachable; must not fall back to a built-in secret")
	}
}

func TestLoadAPIKeyFailsOnBadToken(t *testing.T) {
	v := fakeVault(t)
	defer v.Close()

	if _, _, err := LoadAPIKey(NewSecretStore(v.URL, "wrong-token")); err == nil {
		t.Fatal("expected an error for a rejected token")
	}
}

func TestOutboundCallsHaveATimeout(t *testing.T) {
	if NewSecretStore("http://x", "t").client.Timeout == 0 {
		t.Fatal("the Vault HTTP client has no timeout; a hung Vault would hang the service")
	}
}

func TestHealth(t *testing.T) {
	rec := httptest.NewRecorder()
	NewServer(testKey, "vault").Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/health", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestConfigCheckNeverReturnsTheSecret(t *testing.T) {
	rec := httptest.NewRecorder()
	NewServer(testKey, "vault").Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/config/check", nil))

	if strings.Contains(rec.Body.String(), testKey) {
		t.Fatalf("response leaks the secret: %s", rec.Body.String())
	}
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body["api_key_loaded"] != true || body["source"] != "vault" {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestNotifyAccepted(t *testing.T) {
	h := NewServer(testKey, "vault").Handler()

	rec := post(t, h, "/notify", `{"to":"+919999999999","message":"hello"}`)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status %d, want 202", rec.Code)
	}
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body["id"] != float64(1) || body["status"] != "queued" || body["key_last4"] != "WXYZ" {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestNotifyRejectsBadInput(t *testing.T) {
	h := NewServer(testKey, "vault").Handler()
	cases := map[string]string{
		"missing to":      `{"message":"hi"}`,
		"blank to":        `{"to":"   ","message":"hi"}`,
		"missing message": `{"to":"+91999"}`,
		"blank message":   `{"to":"+91999","message":"  "}`,
		"not json":        `nope`,
	}
	for name, body := range cases {
		if rec := post(t, h, "/notify", body); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status %d, want 400", name, rec.Code)
		}
	}
}

func TestNotifyIDsIncrease(t *testing.T) {
	h := NewServer(testKey, "vault").Handler()
	a := post(t, h, "/notify", `{"to":"1","message":"a"}`)
	b := post(t, h, "/notify", `{"to":"2","message":"b"}`)
	var x, y map[string]any
	_ = json.Unmarshal(a.Body.Bytes(), &x)
	_ = json.Unmarshal(b.Body.Bytes(), &y)
	if x["id"] != float64(1) || y["id"] != float64(2) {
		t.Fatalf("ids %v, %v; want 1, 2", x["id"], y["id"])
	}
}
