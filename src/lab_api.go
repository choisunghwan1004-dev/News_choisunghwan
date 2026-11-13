package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const maxDataset = 20 << 20

func fail(w http.ResponseWriter, code int, message string) {
	w.WriteHeader(code)
	serveJSON(w, map[string]any{"error": message})
}
func postOnly(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodPost {
		fail(w, 405, "POST만 지원합니다.")
		return false
	}
	return true
}
func decodeJSON(r *http.Request, target any) error {
	return json.NewDecoder(io.LimitReader(r.Body, 2<<20)).Decode(target)
}
func readUpload(r *http.Request, field string, max int64) ([]byte, string, error) {
	r.Body = http.MaxBytesReader(nil, r.Body, max+(1<<20))
	f, h, e := r.FormFile(field)
	if e != nil {
		return nil, "", fmt.Errorf("파일을 선택하세요: %w", e)
	}
	defer f.Close()
	b, e := io.ReadAll(io.LimitReader(f, max+1))
	if e != nil {
		return nil, "", e
	}
	if int64(len(b)) > max {
		return nil, "", fmt.Errorf("파일 제한 %d MB 초과", max>>20)
	}
	if !utf8.Valid(b) && (strings.HasSuffix(strings.ToLower(h.Filename), ".csv") || strings.HasSuffix(strings.ToLower(h.Filename), ".jsonl")) {
		return nil, "", errors.New("데이터 파일은 UTF-8로 저장해 주세요. CP949라면 UTF-8로 변환해 다시 올리세요.")
	}
	return bytes.TrimPrefix(b, []byte{0xef, 0xbb, 0xbf}), h.Filename, nil
}

type columnSummary struct {
	Name         string         `json:"name"`
	Count        int            `json:"count"`
	Missing      int            `json:"missing"`
	NumericCount int            `json:"numericCount"`
	Min          float64        `json:"min"`
	Max          float64        `json:"max"`
	Mean         float64        `json:"mean"`
	Median       float64        `json:"median"`
	Unique       int            `json:"unique"`
	Top          []valueCount   `json:"top"`
	Terms        []valueCount   `json:"terms"`
	TermCounts   map[string]int `json:"-"`
	Values       []float64      `json:"-"`
	Counts       map[string]int `json:"-"`
	Sum          float64        `json:"-"`
}

var newsStopwords = map[string]bool{"가상": true, "뉴스": true, "데이터": true, "기사": true, "에서": true, "합니다": true, "대한": true, "그리고": true, "오늘": true, "조사": true, "기록": true, "발표": true}

type valueCount struct {
	Value string `json:"value"`
	Count int    `json:"count"`
}

func parseRows(b []byte, filename string) ([]string, [][]string, error) {
	isJSONL := strings.HasSuffix(strings.ToLower(filename), ".jsonl")
	if isJSONL {
		lines := bytes.Split(b, []byte{'\n'})
		objects := make([]map[string]any, 0, len(lines))
		keySet := map[string]bool{}
		for i, line := range lines {
			line = bytes.TrimSpace(line)
			if len(line) == 0 {
				continue
			}
			var obj map[string]any
			decoder := json.NewDecoder(bytes.NewReader(line))
			decoder.UseNumber()
			if e := decoder.Decode(&obj); e != nil {
				return nil, nil, fmt.Errorf("JSONL %d행 오류: %w", i+1, e)
			}
			for k := range obj {
				keySet[k] = true
			}
			objects = append(objects, obj)
			if len(objects) > 200000 {
				return nil, nil, errors.New("최대 20만 행까지 분석합니다.")
			}
		}
		if len(objects) == 0 || len(keySet) == 0 {
			return nil, nil, errors.New("JSONL 파일이 비어 있습니다. 객체 데이터가 한 행 이상 필요합니다.")
		}
		keys := make([]string, 0, len(keySet))
		for k := range keySet {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		rows := make([][]string, 0, len(objects))
		for _, o := range objects {
			row := make([]string, len(keys))
			for i, k := range keys {
				if v, ok := o[k]; ok && v != nil {
					row[i] = fmt.Sprint(v)
				}
			}
			rows = append(rows, row)
		}
		return keys, rows, nil
	}
	cr := csv.NewReader(bytes.NewReader(b))
	cr.FieldsPerRecord = -1
	cr.LazyQuotes = true
	cr.TrimLeadingSpace = true
	all, e := cr.ReadAll()
	if e != nil {
		return nil, nil, e
	}
	if len(all) < 2 {
		return nil, nil, errors.New("CSV에는 열 제목과 데이터 행이 필요합니다.")
	}
	if len(all) > 200001 {
		return nil, nil, errors.New("최대 20만 행까지 분석합니다.")
	}
	return all[0], all[1:], nil
}
func summarize(headers []string, rows [][]string) map[string]any {
	cols := make([]*columnSummary, len(headers))
	for i, h := range headers {
		cols[i] = &columnSummary{Name: h, Counts: map[string]int{}, TermCounts: map[string]int{}, Values: make([]float64, 0)}
	}
	previews := make([]map[string]string, 0, 5)
	for ix, row := range rows {
		if ix < 5 {
			v := map[string]string{}
			for j, h := range headers {
				if j < len(row) {
					v[h] = row[j]
				}
			}
			previews = append(previews, v)
		}
		for j, c := range cols {
			v := ""
			if j < len(row) {
				v = strings.TrimSpace(row[j])
			}
			if v == "" {
				c.Missing++
				continue
			}
			c.Count++
			if len(c.Counts) < 6000 || c.Counts[v] > 0 {
				c.Counts[v]++
			}
			if len(v) > 4 && len(v) < 900 {
				for _, word := range tokenize(v, false) {
					if (len([]rune(word)) >= 2 && len(c.TermCounts) < 4500 || c.TermCounts[word] > 0) && !newsStopwords[word] {
						c.TermCounts[word]++
					}
				}
			}
			if n, e := strconv.ParseFloat(strings.ReplaceAll(v, ",", ""), 64); e == nil && !math.IsNaN(n) && !math.IsInf(n, 0) {
				c.NumericCount++
				c.Sum += n
				c.Values = append(c.Values, n)
				if c.NumericCount == 1 || n < c.Min {
					c.Min = n
				}
				if c.NumericCount == 1 || n > c.Max {
					c.Max = n
				}
			}
		}
	}
	for _, c := range cols {
		c.Unique = len(c.Counts)
		if c.NumericCount > 0 {
			c.Mean = c.Sum / float64(c.NumericCount)
			sort.Float64s(c.Values)
			n := len(c.Values)
			if n%2 == 1 {
				c.Median = c.Values[n/2]
			} else {
				c.Median = (c.Values[n/2-1] + c.Values[n/2]) / 2
			}
		}
		pairs := make([]valueCount, 0, len(c.Counts))
		for v, n := range c.Counts {
			pairs = append(pairs, valueCount{v, n})
		}
		sort.Slice(pairs, func(i, j int) bool {
			if pairs[i].Count == pairs[j].Count {
				return pairs[i].Value < pairs[j].Value
			}
			return pairs[i].Count > pairs[j].Count
		})
		if len(pairs) > 12 {
			pairs = pairs[:12]
		}
		c.Top = pairs
		terms := make([]valueCount, 0, len(c.TermCounts))
		for v, n := range c.TermCounts {
			terms = append(terms, valueCount{v, n})
		}
		sort.Slice(terms, func(i, j int) bool {
			if terms[i].Count == terms[j].Count {
				return terms[i].Value < terms[j].Value
			}
			return terms[i].Count > terms[j].Count
		})
		if len(terms) > 18 {
			terms = terms[:18]
		}
		c.Terms = terms
	}
	return map[string]any{"totalRows": len(rows), "columnCount": len(headers), "columns": cols, "preview": previews, "note": "입력 파일의 숫자와 텍스트를 계산한 결과입니다. 자료 출처와 정확성은 검증하지 않았습니다."}
}
func analyzeHandler(w http.ResponseWriter, r *http.Request) {
	if !postOnly(w, r) {
		return
	}
	b, name, e := readUpload(r, "file", maxDataset)
	if e != nil {
		fail(w, 400, e.Error())
		return
	}
	h, rows, e := parseRows(b, name)
	if e != nil {
		fail(w, 400, e.Error())
		return
	}
	if len(h) > 100 {
		fail(w, 400, "최대 100개 열까지 분석합니다.")
		return
	}
	serveJSON(w, summarize(h, rows))
}

// Web crawling enforces public host resolution, limited traversal, response caps, and robots.txt.
var titleRE = regexp.MustCompile(`(?is)<title\b[^>]*>(.*?)</title>`)
var h1RE = regexp.MustCompile(`(?is)<h1\b[^>]*>(.*?)</h1>`)
var paraRE = regexp.MustCompile(`(?is)<(?:p|blockquote)\b[^>]*>(.*?)</(?:p|blockquote)>`)
var linkRE = regexp.MustCompile(`(?is)<a\s+[^>]*href\s*=\s*["']([^"']+)["'][^>]*>`)
var stripRE = regexp.MustCompile(`(?is)<[^>]+>`)
var killRE = regexp.MustCompile(`(?is)<(?:script|style|noscript|svg)\b[^>]*>.*?</(?:script|style|noscript|svg)>`)
var spaceRE = regexp.MustCompile(`\s+`)

func cleanHTML(raw string) string {
	raw = killRE.ReplaceAllString(raw, "")
	raw = stripRE.ReplaceAllString(raw, " ")
	return strings.TrimSpace(spaceRE.ReplaceAllString(html.UnescapeString(raw), " "))
}

type fetchedPage struct {
	URL         string   `json:"url"`
	Title       string   `json:"title"`
	Heading     string   `json:"heading"`
	Text        string   `json:"text"`
	CollectedAt string   `json:"collected_at"`
	Links       []string `json:"-"`
}

func extractPage(raw string, u string) fetchedPage {
	get := func(re *regexp.Regexp) string {
		m := re.FindStringSubmatch(raw)
		if len(m) > 1 {
			return cleanHTML(m[1])
		}
		return ""
	}
	p := fetchedPage{URL: u, Title: get(titleRE), Heading: get(h1RE), CollectedAt: time.Now().Format(time.RFC3339)}
	lines := []string{}
	for _, m := range paraRE.FindAllStringSubmatch(raw, 200) {
		s := cleanHTML(m[1])
		if len([]rune(s)) >= 12 {
			lines = append(lines, s)
		}
	}
	p.Text = strings.Join(lines, "\n")
	if len([]rune(p.Text)) > 7000 {
		p.Text = string([]rune(p.Text)[:7000])
	}
	base, _ := url.Parse(u)
	seen := map[string]bool{}
	for _, m := range linkRE.FindAllStringSubmatch(raw, 120) {
		u2, e := url.Parse(html.UnescapeString(m[1]))
		if e != nil {
			continue
		}
		absolute := base.ResolveReference(u2)
		absolute.Fragment = ""
		if absolute.Hostname() == base.Hostname() && absolute.Scheme == base.Scheme && !seen[absolute.String()] {
			seen[absolute.String()] = true
			p.Links = append(p.Links, absolute.String())
		}
	}
	return p
}
func validPublicURL(raw string) error {
	u, e := url.Parse(raw)
	if e != nil {
		return errors.New("URL이 올바르지 않습니다.")
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return errors.New("HTTP/HTTPS URL만 허용합니다.")
	}
	if u.User != nil || u.Hostname() == "" {
		return errors.New("사용자 정보나 잘못된 호스트는 허용하지 않습니다.")
	}
	host := strings.ToLower(u.Hostname())
	if host == "localhost" || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local") || strings.HasSuffix(host, ".internal") {
		return errors.New("내부 네트워크 주소는 크롤링할 수 없습니다.")
	}
	if p := u.Port(); p != "" && p != "80" && p != "443" {
		return errors.New("80/443 포트만 허용합니다.")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ips, e := net.DefaultResolver.LookupIPAddr(ctx, host)
	if e != nil || len(ips) == 0 {
		return fmt.Errorf("도메인 확인 실패: %s", host)
	}
	for _, entry := range ips {
		ip := entry.IP
		if !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			return errors.New("비공개 IP 주소로 연결할 수 없습니다.")
		}
	}
	return nil
}
func makeCrawlClient() *http.Client {
	tr := &http.Transport{Proxy: nil, DisableKeepAlives: true, MaxIdleConns: 0, DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, e := net.SplitHostPort(addr)
		if e != nil {
			return nil, e
		}
		if e = validPublicURL("https://" + net.JoinHostPort(host, port)); e != nil {
			return nil, e
		}
		ips, e := net.DefaultResolver.LookupIPAddr(ctx, host)
		if e != nil || len(ips) == 0 {
			return nil, errors.New("DNS lookup failed")
		}
		for _, ip := range ips {
			if ip.IP.IsGlobalUnicast() && !ip.IP.IsPrivate() && !ip.IP.IsLoopback() && !ip.IP.IsLinkLocalUnicast() {
				return (&net.Dialer{Timeout: 6 * time.Second}).DialContext(ctx, network, net.JoinHostPort(ip.IP.String(), port))
			}
		}
		return nil, errors.New("only public addresses allowed")
	}}
	return &http.Client{Transport: tr, Timeout: 12 * time.Second, CheckRedirect: func(r *http.Request, via []*http.Request) error {
		if len(via) > 3 {
			return errors.New("리디렉션 최대 3회 초과")
		}
		if len(via) > 0 && r.URL.Hostname() != via[0].URL.Hostname() {
			return errors.New("다른 도메인으로의 리디렉션은 차단됩니다.")
		}
		return validPublicURL(r.URL.String())
	}}
}
func robotsPath(u *url.URL) string {
	p := u.EscapedPath()
	if p == "" {
		return "/"
	}
	return p
}
func robotsPermission(client *http.Client, u *url.URL) error {
	robotsURL := u.Scheme + "://" + u.Host + "/robots.txt"
	req, _ := http.NewRequest(http.MethodGet, robotsURL, nil)
	req.Header.Set("User-Agent", "DataJournalismLearningLab/2.0 (+educational; respectful-crawl)")
	resp, e := client.Do(req)
	if e != nil {
		return fmt.Errorf("robots.txt 확인 실패: %w", e)
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return nil
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("robots.txt 상태 %d: 확인 불가로 수집 보류", resp.StatusCode)
	}
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 256<<10))
	enabled := false
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(strings.SplitN(line, "#", 2)[0])
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key, val := strings.ToLower(strings.TrimSpace(parts[0])), strings.TrimSpace(parts[1])
		if key == "user-agent" {
			enabled = val == "*" || strings.Contains(val, "DataJournalismLearningLab")
			continue
		}
		if key == "disallow" && enabled && val != "" && strings.HasPrefix(robotsPath(u), val) {
			return errors.New("사이트 robots.txt에서 해당 경로 수집을 금지했습니다.")
		}
	}
	return nil
}
func crawlHandler(w http.ResponseWriter, r *http.Request) {
	if !postOnly(w, r) {
		return
	}
	var req struct {
		URL      string `json:"url"`
		MaxPages int    `json:"maxPages"`
		Demo     bool   `json:"demo"`
	}
	if e := decodeJSON(r, &req); e != nil {
		fail(w, 400, "요청 JSON 오류")
		return
	}
	if req.Demo {
		raw := `<html><head><title>교육용 지역 뉴스</title></head><body><h1>가상 데이터 저널리즘 실습</h1><p>이 문서는 교육용 가상 뉴스입니다. 실제 사건을 보도하는 자료가 아닙니다.</p><p>따릉이 연간 이용 4500만 건, 직장인 점심값 11000원, 공연 관람객 중 20대 41퍼센트는 모두 실습 수치입니다.</p><blockquote>실제 보도 전에는 반드시 원문에서 숫자를 재확인해야 합니다.</blockquote></body></html>`
		serveJSON(w, map[string]any{"pages": []fetchedPage{extractPage(raw, "demo://fictional-example")}, "note": "오프라인 내장 HTML로 추출 로직을 실행했습니다. 실제 인터넷 수집 결과가 아닙니다."})
		return
	}
	if req.MaxPages < 1 || req.MaxPages > 3 {
		req.MaxPages = 1
	}
	if e := validPublicURL(req.URL); e != nil {
		fail(w, 400, e.Error())
		return
	}
	start, _ := url.Parse(req.URL)
	client := makeCrawlClient()
	if e := robotsPermission(client, start); e != nil {
		fail(w, 400, e.Error())
		return
	}
	queue := []string{req.URL}
	visited := map[string]bool{}
	pages := []fetchedPage{}
	for len(queue) > 0 && len(pages) < req.MaxPages {
		target := queue[0]
		queue = queue[1:]
		if visited[target] {
			continue
		}
		visited[target] = true
		if e := validPublicURL(target); e != nil {
			continue
		}
		u, _ := url.Parse(target)
		if u.Hostname() != start.Hostname() {
			continue
		}
		if e := robotsPermission(client, u); e != nil {
			continue
		}
		rq, _ := http.NewRequestWithContext(r.Context(), "GET", target, nil)
		rq.Header.Set("User-Agent", "DataJournalismLearningLab/2.0 educational research")
		resp, e := client.Do(rq)
		if e != nil {
			if len(pages) == 0 {
				fail(w, 502, "웹 접속 실패: "+e.Error())
				return
			}
			continue
		}
		if resp.StatusCode != 200 {
			resp.Body.Close()
			if len(pages) == 0 {
				fail(w, 502, fmt.Sprintf("응답 코드: %d", resp.StatusCode))
				return
			}
			continue
		}
		if !strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "text/html") {
			resp.Body.Close()
			if len(pages) == 0 {
				fail(w, 415, "HTML 페이지가 아닙니다.")
				return
			}
			continue
		}
		bytes, e := io.ReadAll(io.LimitReader(resp.Body, 750<<10))
		resp.Body.Close()
		if e != nil {
			continue
		}
		page := extractPage(string(bytes), resp.Request.URL.String())
		if page.Title == "" && page.Text == "" {
			continue
		}
		pages = append(pages, page)
		for _, link := range page.Links {
			if !visited[link] && len(queue) < 12 {
				queue = append(queue, link)
			}
		}
		if len(pages) < req.MaxPages {
			select {
			case <-r.Context().Done():
				return
			case <-time.After(800 * time.Millisecond):
			}
		}
	}
	if len(pages) == 0 {
		fail(w, 422, "추출할 텍스트가 없거나 사이트에서 수집을 제한했습니다.")
		return
	}
	serveJSON(w, map[string]any{"pages": pages, "note": "공개 HTML 일부만 추출했습니다. 출처·이용 약관·저작권과 문장 정확성은 직접 확인하세요. 기사 전문 재게시를 권장하지 않습니다."})
}

// Real supervised text-model training and held-out evaluation (not an LLM/LoRA fine-tune).
type textExample struct {
	Text  string `json:"text"`
	Label string `json:"label"`
}
type textTrainReq struct {
	Examples []textExample `json:"examples"`
	Predict  string        `json:"predict"`
}
type nbModel struct {
	Classes []string
	Vocab   map[string]bool
	Terms   map[string]map[string]int
	Totals  map[string]int
	Priors  map[string]int
	Alpha   float64
	Bigrams bool
	TrainN  int
}

func tokenize(text string, bigrams bool) []string {
	out := []string{}
	buf := []rune{}
	flush := func() {
		if len(buf) > 0 {
			out = append(out, strings.ToLower(string(buf)))
			buf = buf[:0]
		}
	}
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			buf = append(buf, r)
		} else {
			flush()
		}
	}
	flush()
	if bigrams {
		n := len(out)
		for i := 0; i+1 < n; i++ {
			out = append(out, out[i]+"_"+out[i+1])
		}
	}
	return out
}
func fitModel(ex []textExample, alpha float64, bigram bool) *nbModel {
	m := &nbModel{Vocab: map[string]bool{}, Terms: map[string]map[string]int{}, Totals: map[string]int{}, Priors: map[string]int{}, Alpha: alpha, Bigrams: bigram, TrainN: len(ex)}
	for _, r := range ex {
		if m.Terms[r.Label] == nil {
			m.Terms[r.Label] = map[string]int{}
			m.Classes = append(m.Classes, r.Label)
		}
		m.Priors[r.Label]++
		for _, term := range tokenize(r.Text, bigram) {
			m.Terms[r.Label][term]++
			m.Totals[r.Label]++
			m.Vocab[term] = true
		}
	}
	sort.Strings(m.Classes)
	return m
}
func (m *nbModel) predict(text string) string {
	winner := ""
	score := -math.MaxFloat64
	for _, label := range m.Classes {
		prior := math.Log(float64(m.Priors[label]) / float64(m.TrainN))
		s := prior
		denom := float64(m.Totals[label]) + m.Alpha*float64(len(m.Vocab))
		for _, term := range tokenize(text, m.Bigrams) {
			if m.Vocab[term] {
				s += math.Log((float64(m.Terms[label][term]) + m.Alpha) / denom)
			}
		}
		if winner == "" || s > score {
			winner, score = label, s
		}
	}
	return winner
}
func evalModel(m *nbModel, data []textExample) map[string]any {
	correct := 0
	per := map[string]map[string]int{}
	for _, e := range data {
		if per[e.Label] == nil {
			per[e.Label] = map[string]int{}
		}
		prediction := m.predict(e.Text)
		per[e.Label][prediction]++
		if prediction == e.Label {
			correct++
		}
	}
	acc := 0.0
	if len(data) > 0 {
		acc = 100 * float64(correct) / float64(len(data))
	}
	return map[string]any{"correct": correct, "total": len(data), "accuracy": math.Round(acc*10) / 10, "confusion": per}
}
func splitExamples(ex []textExample) (train, val, test []textExample) {
	groups := map[string][]textExample{}
	labels := []string{}
	for _, e := range ex {
		if _, ok := groups[e.Label]; !ok {
			labels = append(labels, e.Label)
		}
		groups[e.Label] = append(groups[e.Label], e)
	}
	sort.Strings(labels)
	for _, l := range labels {
		g := groups[l]
		for i, e := range g {
			switch {
			case i%5 == 3:
				val = append(val, e)
			case i%5 == 4:
				test = append(test, e)
			default:
				train = append(train, e)
			}
		}
	}
	return
}
func trainHandler(w http.ResponseWriter, r *http.Request) {
	if !postOnly(w, r) {
		return
	}
	var req textTrainReq
	if e := decodeJSON(r, &req); e != nil {
		fail(w, 400, "학습 데이터 JSON 오류")
		return
	}
	if len(req.Examples) < 12 || len(req.Examples) > 1000 {
		fail(w, 400, "학습 행은 12~1000개가 필요합니다.")
		return
	}
	counts := map[string]int{}
	for _, ex := range req.Examples {
		ex.Label = strings.TrimSpace(ex.Label)
		if ex.Label == "" || strings.TrimSpace(ex.Text) == "" {
			fail(w, 400, "빈 레이블 또는 텍스트가 있습니다.")
			return
		}
		counts[ex.Label]++
	}
	if len(counts) < 2 || len(counts) > 10 {
		fail(w, 400, "분류 레이블은 2~10종류여야 합니다.")
		return
	}
	for label, c := range counts {
		if c < 5 {
			fail(w, 400, label+": 레이블당 5개 이상 필요합니다.")
			return
		}
	}
	train, val, test := splitExamples(req.Examples)
	base := fitModel(train, 1, false)
	baseVal := evalModel(base, val)
	best := fitModel(train, 0.5, true)
	bestVal := evalModel(best, val)
	bestChoice := "bigrams α=0.5"
	candidates := []struct {
		a    float64
		b    bool
		name string
	}{{0.5, false, "unigrams α=0.5"}, {1, true, "bigrams α=1"}, {2, true, "bigrams α=2"}, {1, false, "unigrams α=1"}}
	for _, cfg := range candidates {
		m := fitModel(train, cfg.a, cfg.b)
		score := evalModel(m, val)["accuracy"].(float64)
		if score > bestVal["accuracy"].(float64) {
			best, bestVal, bestChoice = m, evalModel(m, val), cfg.name
		}
	}
	prediction := ""
	if strings.TrimSpace(req.Predict) != "" {
		prediction = best.predict(req.Predict)
	}
	serveJSON(w, map[string]any{"type": "나이브 베이즈 텍스트 분류 모델 훈련 및 하이퍼파라미터 튜닝", "notLLM": true, "split": map[string]int{"train": len(train), "validation": len(val), "test": len(test)}, "baseline": map[string]any{"parameter": "unigrams α=1", "validation": baseVal, "test": evalModel(base, test)}, "tuned": map[string]any{"parameter": bestChoice, "validation": bestVal, "test": evalModel(best, test)}, "prediction": prediction, "vocabularySize": len(best.Vocab), "note": "검증 세트에서 매개변수를 선택하고 별도 테스트 세트로 평가했습니다. LLM LoRA/Unsloth 파인튜닝은 원본 GPU 노트북을 별도 환경에서 실행해야 합니다."})
}

func transcribeHandler(a *app, w http.ResponseWriter, r *http.Request) {
	if !postOnly(w, r) {
		return
	}
	python := filepath.Join(a.root, "_voice_venv", "Scripts", "python.exe")
	if _, e := os.Stat(python); e != nil {
		fail(w, 409, "음성 인식 환경이 설치되지 않았습니다. 04_음성인식_설치.bat를 먼저 실행하세요.")
		return
	}
	b, name, e := readUpload(r, "file", 25<<20)
	if e != nil {
		fail(w, 400, e.Error())
		return
	}
	ext := strings.ToLower(filepath.Ext(name))
	allowed := map[string]bool{".wav": true, ".mp3": true, ".m4a": true, ".flac": true, ".ogg": true}
	if !allowed[ext] {
		fail(w, 400, "WAV, MP3, M4A, FLAC, OGG만 지원합니다.")
		return
	}
	tmp, e := os.MkdirTemp("", "dj_voice_")
	if e != nil {
		fail(w, 500, e.Error())
		return
	}
	defer os.RemoveAll(tmp)
	audio := filepath.Join(tmp, "audio"+ext)
	if e = os.WriteFile(audio, b, 0600); e != nil {
		fail(w, 500, e.Error())
		return
	}
	runner := filepath.Join(a.root, "tools", "transcribe_local.py")
	ctx, cancel := context.WithTimeout(r.Context(), 4*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, python, runner, audio)
	cmd.Env = append(os.Environ(), "PYTHONIOENCODING=utf-8")
	out, e := cmd.CombinedOutput()
	if e != nil {
		msg := string(out)
		if len(msg) > 800 {
			msg = msg[len(msg)-800:]
		}
		fail(w, 503, "음성 인식 실패: "+msg)
		return
	}
	var result map[string]any
	if e = json.Unmarshal(out, &result); e != nil {
		fail(w, 500, "전사 결과 파싱 오류")
		return
	}
	serveJSON(w, result)
}
func (a *app) registerLabRoutes(m *http.ServeMux) {
	m.HandleFunc("/api/analyze", analyzeHandler)
	m.HandleFunc("/api/crawl", crawlHandler)
	m.HandleFunc("/api/train", trainHandler)
	m.HandleFunc("/api/transcribe", func(w http.ResponseWriter, r *http.Request) { transcribeHandler(a, w, r) })
}
