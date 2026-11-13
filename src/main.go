package main

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

//go:embed web/*
var embeddedWeb embed.FS

const appVersion = "2.0.1"

type Notebook struct {
	Name         string `json:"name"`
	Group        string `json:"group"`
	RelativePath string `json:"relativePath"`
	FileSize     int64  `json:"fileSize"`
}

type app struct {
	root          string
	orig          string
	notebooks     []Notebook
	notebookFiles map[string]string
	closeServer   func()
}

func rootDir() string {
	if specified := os.Getenv("DJ_STUDIO_ROOT"); specified != "" {
		return specified
	}
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exe)
}

func newApp(root string) *app {
	a := &app{root: root, orig: filepath.Join(root, "original", "ai-datajournalism-labs-main"), notebookFiles: map[string]string{}}
	// The original repository is immutable. Inspect it only for notebook links.
	_ = filepath.WalkDir(a.orig, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d == nil || d.IsDir() || !strings.EqualFold(filepath.Ext(d.Name()), ".ipynb") {
			return nil
		}
		rel, e := filepath.Rel(a.orig, p)
		if e != nil {
			return nil
		}
		clean := filepath.ToSlash(rel)
		group := filepath.ToSlash(filepath.Dir(rel))
		info, e := d.Info()
		if e != nil {
			return nil
		}
		a.notebooks = append(a.notebooks, Notebook{Name: d.Name(), Group: group, RelativePath: clean, FileSize: info.Size()})
		a.notebookFiles[clean] = p
		return nil
	})
	sort.Slice(a.notebooks, func(i, j int) bool { return a.notebooks[i].RelativePath < a.notebooks[j].RelativePath })
	return a
}

func (a *app) handler() http.Handler {
	m := http.NewServeMux()
	staticSub, _ := fs.Sub(embeddedWeb, "web")
	m.Handle("/", http.FileServer(http.FS(staticSub)))
	m.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		serveJSON(w, map[string]any{"ok": true, "version": appVersion, "offline": true, "notebooks": len(a.notebooks), "modules": []string{"visualization", "bigdata", "crawler", "ml_classification", "breaking_alerts", "voice_transcription_optional"}})
	})
	m.HandleFunc("/api/labs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		serveJSON(w, map[string]any{"notebooks": a.notebooks, "originalAvailable": len(a.notebooks) > 0})
	})
	m.HandleFunc("/api/notebook", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		key := r.URL.Query().Get("path")
		path, ok := a.notebookFiles[key]
		if !ok {
			http.Error(w, "notebook not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filepath.Base(path)))
		w.Header().Set("Content-Type", "application/x-ipynb+json; charset=utf-8")
		http.ServeFile(w, r, path)
	})
	a.registerLabRoutes(m)
	m.HandleFunc("/api/open-folder", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var err error
		switch runtime.GOOS {
		case "windows":
			err = exec.Command("explorer.exe", a.orig).Start()
		case "darwin":
			err = exec.Command("open", a.orig).Start()
		default:
			err = exec.Command("xdg-open", a.orig).Start()
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		serveJSON(w, map[string]any{"ok": true})
	})
	m.HandleFunc("/api/start-jupyter", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if runtime.GOOS != "windows" {
			http.Error(w, "Jupyter launcher is Windows-only", 501)
			return
		}
		py := filepath.Join(a.root, "_venv", "Scripts", "python.exe")
		if _, err := os.Stat(py); err != nil {
			http.Error(w, "02_노트북_환경설치.bat를 먼저 실행하세요.", 409)
			return
		}
		batch := filepath.Join(a.root, "03_노트북_열기.bat")
		cmd := exec.Command("cmd.exe", "/c", batch)
		cmd.Dir = a.root
		if err := cmd.Start(); err != nil {
			http.Error(w, err.Error(), 503)
			return
		}
		serveJSON(w, map[string]any{"ok": true, "message": "Jupyter 실행을 요청했습니다. 별도 창과 브라우저에서 확인하세요."})
	})
	m.HandleFunc("/api/exit", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		serveJSON(w, map[string]any{"ok": true, "message": "프로그램을 종료했습니다."})
		if a.closeServer != nil {
			go a.closeServer()
		}
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The service is local-only, and mutation endpoints reject foreign Origins.
		if r.Method == http.MethodPost {
			origin := r.Header.Get("Origin")
			expected := "http://" + r.Host
			if origin != "" && origin != expected {
				http.Error(w, "cross-origin request blocked", http.StatusForbidden)
				return
			}
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Cache-Control", "no-store")
		m.ServeHTTP(w, r)
	})
}

func serveJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(v)
}
func openBrowser(u string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", u).Start()
	case "darwin":
		return exec.Command("open", u).Start()
	default:
		return exec.Command("xdg-open", u).Start()
	}
}
func main() {
	a := newApp(rootDir())
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		log.Fatal(err)
	}
	server := &http.Server{Handler: a.handler(), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 45 * time.Second}
	a.closeServer = func() { _ = server.Close() }
	url := fmt.Sprintf("http://127.0.0.1:%d/", ln.Addr().(*net.TCPAddr).Port)
	fmt.Println("데이터 저널리즘 뉴스 스튜디오", appVersion)
	fmt.Println("접속 주소:", url)
	fmt.Printf("노트북 %d개 발견\n", len(a.notebooks))
	if os.Getenv("DJ_NO_BROWSER") != "1" {
		if e := openBrowser(url); e != nil {
			fmt.Println("브라우저 자동 실행 실패. 위 주소를 직접 여세요:", e)
		}
	}
	err = server.Serve(ln)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
