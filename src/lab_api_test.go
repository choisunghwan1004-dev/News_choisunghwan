package main

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func multipartRequest(t *testing.T, path, name, contents string) *http.Request {
	t.Helper()
	var b bytes.Buffer
	m := multipart.NewWriter(&b)
	f, e := m.CreateFormFile("file", name)
	if e != nil {
		t.Fatal(e)
	}
	_, _ = f.Write([]byte(contents))
	m.Close()
	r := httptest.NewRequest("POST", path, &b)
	r.Header.Set("Content-Type", m.FormDataContentType())
	return r
}
func TestV2AnalysisAndMissing(t *testing.T) {
	a := newApp(t.TempDir())
	rec := httptest.NewRecorder()
	a.handler().ServeHTTP(rec, multipartRequest(t, "/api/analyze", "values.csv", "section,value\n교통,10\n교통,20\n문화,\n"))
	if rec.Code != 200 {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	var obj struct {
		TotalRows int             `json:"totalRows"`
		Columns   []columnSummary `json:"columns"`
	}
	if e := json.Unmarshal(rec.Body.Bytes(), &obj); e != nil {
		t.Fatal(e)
	}
	if obj.TotalRows != 3 || obj.Columns[1].Mean != 15 || obj.Columns[1].Missing != 1 {
		t.Fatalf("wrong stats %+v", obj)
	}
}
func TestV2JSONL(t *testing.T) {
	a := newApp(t.TempDir())
	rec := httptest.NewRecorder()
	a.handler().ServeHTTP(rec, multipartRequest(t, "/api/analyze", "items.jsonl", "{\"title\":\"뉴스 A\",\"count\":5}\n{\"title\":\"뉴스 B\",\"count\":8}\n"))
	if rec.Code != 200 {
		t.Fatalf("code=%d %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "뉴스 A") {
		t.Fatal("missing JSONL data")
	}
}
func TestV2CrawlDemoAndBlockPrivate(t *testing.T) {
	a := newApp(t.TempDir())
	h := a.handler()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("POST", "/api/crawl", strings.NewReader(`{"demo":true}`)))
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "실습") {
		t.Fatalf("demo=%d %s", rec.Code, rec.Body.String())
	}
	for _, domain := range []string{"http://127.0.0.1/", "http://localhost:3000/", "file:///etc/passwd"} {
		rec = httptest.NewRecorder()
		payload, _ := json.Marshal(map[string]string{"url": domain})
		h.ServeHTTP(rec, httptest.NewRequest("POST", "/api/crawl", bytes.NewReader(payload)))
		if rec.Code == 200 {
			t.Fatalf("private URL allowed %s", domain)
		}
	}
}
func TestV2MLTrainAndHeldout(t *testing.T) {
	samples := []textExample{}
	for _, category := range []string{"경제", "문화", "교통"} {
		for i := 0; i < 15; i++ {
			samples = append(samples, textExample{Text: category + " 뉴스를 조사했습니다 " + string(rune('가'+i)), Label: category})
		}
	}
	data, _ := json.Marshal(textTrainReq{Examples: samples, Predict: "경제 뉴스"})
	a := newApp(t.TempDir())
	rec := httptest.NewRecorder()
	a.handler().ServeHTTP(rec, httptest.NewRequest("POST", "/api/train", bytes.NewReader(data)))
	if rec.Code != 200 {
		t.Fatalf("train=%d %s", rec.Code, rec.Body.String())
	}
	var out struct {
		Split struct {
			Train      int `json:"train"`
			Validation int `json:"validation"`
			Test       int `json:"test"`
		} `json:"split"`
		NotLLM bool `json:"notLLM"`
	}
	if e := json.Unmarshal(rec.Body.Bytes(), &out); e != nil {
		t.Fatal(e)
	}
	if !out.NotLLM || out.Split.Test == 0 || out.Split.Validation == 0 {
		t.Fatalf("invalid train data: %+v", out)
	}
}
func TestV2AudioNeedsOptionalInstall(t *testing.T) {
	a := newApp(t.TempDir())
	rec := httptest.NewRecorder()
	a.handler().ServeHTTP(rec, httptest.NewRequest("POST", "/api/transcribe", strings.NewReader("")))
	if rec.Code != 409 {
		t.Fatalf("expected 409 but %d", rec.Code)
	}
}

// Regression: an empty JSONL must be rejected, not passed to the big-data UI
// where an empty column list causes a client-side panic.
func TestV201RejectEmptyJSONL(t *testing.T) {
	a := newApp(t.TempDir())
	rec := httptest.NewRecorder()
	a.handler().ServeHTTP(rec, multipartRequest(t, "/api/analyze", "empty.jsonl", "\n\n"))
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "비어") {
		t.Fatalf("expected 400 empty-jsonl error, got %d: %s", rec.Code, rec.Body.String())
	}
}

// Regression: crawler output is valid JSONL for the news big-data endpoint.
// This checks the shared data contract without forcing a UI navigation.
func TestV201CrawlToBigdataContract(t *testing.T) {
	a := newApp(t.TempDir())
	rec := httptest.NewRecorder()
	a.handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/crawl", strings.NewReader(`{"demo":true}`)))
	if rec.Code != 200 {
		t.Fatalf("crawl code=%d %s", rec.Code, rec.Body.String())
	}
	var out struct {
		Pages []json.RawMessage `json:"pages"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Pages) != 1 {
		t.Fatalf("crawl pages=%d", len(out.Pages))
	}
	var data bytes.Buffer
	for _, p := range out.Pages {
		data.Write(p)
		data.WriteByte('\n')
	}
	analyzed := httptest.NewRecorder()
	a.handler().ServeHTTP(analyzed, multipartRequest(t, "/api/analyze", "crawled.jsonl", data.String()))
	if analyzed.Code != 200 || !strings.Contains(analyzed.Body.String(), `"totalRows":1`) {
		t.Fatalf("big-data import failed: %d %s", analyzed.Code, analyzed.Body.String())
	}
}
