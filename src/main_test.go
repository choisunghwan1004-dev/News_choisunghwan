package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestHealthAndStatic(t *testing.T) {
	a := newApp(t.TempDir())
	h := a.handler()
	for _, url := range []string{"/api/health", "/", "/app.js", "/style.css"} {
		r := httptest.NewRecorder()
		h.ServeHTTP(r, httptest.NewRequest("GET", url, nil))
		if r.Code != http.StatusOK {
			t.Fatalf("%s -> %d", url, r.Code)
		}
	}
	r := httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest("GET", "/api/health", nil))
	var v map[string]any
	if err := json.Unmarshal(r.Body.Bytes(), &v); err != nil || v["ok"] != true {
		t.Fatalf("bad health: %v %v", v, err)
	}
}
func TestNotebookAllowlist(t *testing.T) {
	root := t.TempDir()
	folder := filepath.Join(root, "original", "ai-datajournalism-labs-main", "safe")
	_ = os.MkdirAll(folder, 0755)
	_ = os.WriteFile(filepath.Join(folder, "test.ipynb"), []byte("{}"), 0644)
	a := newApp(root)
	if len(a.notebooks) != 1 {
		t.Fatalf("notebooks = %d", len(a.notebooks))
	}
	h := a.handler()
	req := httptest.NewRequest("GET", "/api/notebook?path=../../etc/passwd", nil)
	r := httptest.NewRecorder()
	h.ServeHTTP(r, req)
	if r.Code != http.StatusNotFound {
		t.Fatalf("traversal code=%d", r.Code)
	}
}
func TestOriginDefense(t *testing.T) {
	a := newApp(t.TempDir())
	r := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/exit", nil)
	req.Header.Set("Origin", "https://bad.site")
	a.handler().ServeHTTP(r, req)
	if r.Code != 403 {
		t.Fatalf("code=%d", r.Code)
	}
}
