// matrix: 템플릿 × 언어 × 백엔드 매트릭스로 PDF 변환 부하 측정.
//
// 각 (template, language, backend) 조합에 대해:
//   - baseline:   동시성 1, 1 요청  → 순수 처리 시간
//   - concurrent: 동시성 4, 8 요청  → 동시 요청 환경 처리량
//
// 결과는 CSV로 저장하고 콘솔에 요약 표를 출력한다.
package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type combo struct {
	template string
	language string
	path     string
}

func main() {
	var (
		baseURL    = flag.String("url", "http://localhost:8080", "서버 base URL")
		out        = flag.String("out", "reports/matrix.csv", "CSV 출력 파일")
		concurrent = flag.Int("concurrent", 4, "concurrent 모드 동시성")
		concTotal  = flag.Int("concurrent-total", 8, "concurrent 모드 요청 수")
		timeout    = flag.Duration("timeout", 120*time.Second, "요청 타임아웃")
		backendsFl  = flag.String("backends", "gotenberg,weasyprint", "쉼표 구분, 측정할 백엔드. reactpdf 포함 가능 (invoice 시나리오 한정)")
		templatesF  = flag.String("templates", "", "쉼표 구분, 비우면 전체. 예: nested-deep,nested-split")
		languagesF  = flag.String("languages", "", "쉼표 구분, 비우면 전체. 예: ko,ja,en")
		cooldown    = flag.Duration("cooldown", 0, "각 셀 사이 sleep (안정성 확보용)")
		appendCSV   = flag.Bool("append", false, "기존 CSV 에 이어쓰기 (헤더 생략)")
		reactpdfURL = flag.String("reactpdf-url", "http://localhost:5002", "reactpdf 사이드카 base URL (HTTP 서버 우회 직접 호출)")
		dataDir     = flag.String("data-dir", "data", "reactpdf 측정용 Invoice JSON 디렉토리 ({locale}.json)")
	)
	flag.Parse()

	combos := []combo{
		{"invoice", "ko", "output/ko.html"},
		{"invoice", "ja", "output/ja.html"},
		{"invoice", "en", "output/en.html"},
		{"nested-basic", "ko", "web/nested-basic.html"},
		{"nested-basic", "ja", "web/nested-basic-ja.html"},
		{"nested-basic", "en", "web/nested-basic-en.html"},
		{"nested-deep", "ko", "web/nested-deep.html"},
		{"nested-deep", "ja", "web/nested-deep-ja.html"},
		{"nested-deep", "en", "web/nested-deep-en.html"},
		{"nested-split", "ko", "web/nested-split.html"},
		{"nested-split", "ja", "web/nested-split-ja.html"},
		{"nested-split", "en", "web/nested-split-en.html"},
		{"nested-projects", "ko", "web/nested-projects.html"},
		{"nested-projects", "ja", "web/nested-projects-ja.html"},
		{"nested-projects", "en", "web/nested-projects-en.html"},
		{"nested-stress", "ko", "web/nested-stress.html"},
		{"nested-stress", "ja", "web/nested-stress-ja.html"},
		{"nested-stress", "en", "web/nested-stress-en.html"},
	}
	backends := splitCSV(*backendsFl)
	tplFilter := setFromCSV(*templatesF)
	langFilter := setFromCSV(*languagesF)
	if len(tplFilter) > 0 {
		combos = filterCombos(combos, tplFilter, langFilter)
	} else if len(langFilter) > 0 {
		combos = filterCombos(combos, nil, langFilter)
	}

	var f *os.File
	var err error
	if *appendCSV {
		f, err = os.OpenFile(*out, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	} else {
		f, err = os.Create(*out)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "CSV 열기 실패: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	if !*appendCSV {
		w.Write([]string{
			"template", "language", "backend", "size_bytes",
			"baseline_ms",
			"conc_throughput_req_s",
			"conc_avg_ms", "conc_p50_ms", "conc_p95_ms", "conc_max_ms",
			"conc_ok", "conc_fail",
		})
	}

	fmt.Printf("총 %d 조합 × %d 백엔드 = %d 셀\n", len(combos), len(backends), len(combos)*len(backends))
	fmt.Printf("CSV: %s\n\n", *out)

	headerFmt := "%-16s %-3s %-11s %5s | %8s | %7s | %8s %8s %8s %8s | %s\n"
	rowFmt := "%-16s %-3s %-11s %5d | %8s | %7.2f | %8s %8s %8s %8s | %d/%d\n"
	fmt.Printf(headerFmt, "template", "lng", "backend", "kb", "base", "req/s", "avg", "p50", "p95", "max", "ok/fail")
	fmt.Println("-----------------+-----+-----------+------+----------+---------+----------------------------------+--------")

	startAll := time.Now()
	for _, c := range combos {
		htmlBody, err := os.ReadFile(c.path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[skip] %s: %v\n", c.path, err)
			continue
		}
		for _, b := range backends {
			// 백엔드별 입력 분기:
			//   - HTML 백엔드 (gotenberg/weasyprint/docraptor): serve 서버의 /api/convert 로 HTML body POST
			//   - reactpdf:                                    사이드카 :5002/pdf 로 Invoice JSON body 직접 POST
			//                                                  (HTML 전용 nested-* 시나리오는 자동 skip)
			var url, contentType string
			var body []byte
			if b == "reactpdf" {
				if c.template != "invoice" {
					fmt.Fprintf(os.Stderr, "[skip] %s/%s/reactpdf: HTML 전용 시나리오 — Invoice JSON 입력 불가\n", c.template, c.language)
					continue
				}
				dataPath := filepath.Join(*dataDir, c.language+".json")
				jsonBody, rerr := os.ReadFile(dataPath)
				if rerr != nil {
					fmt.Fprintf(os.Stderr, "[skip] %s/%s/reactpdf: %v\n", c.template, c.language, rerr)
					continue
				}
				url = strings.TrimRight(*reactpdfURL, "/") + "/pdf"
				contentType = "application/json"
				body = jsonBody
			} else {
				url = *baseURL + "/api/convert?backend=" + b
				contentType = "text/html; charset=utf-8"
				body = htmlBody
			}

			// baseline: 1 req, sequential
			t0 := time.Now()
			_, code, err := doOne(url, contentType, body, *timeout)
			baseline := time.Since(t0)
			if err != nil || code != 200 {
				fmt.Fprintf(os.Stderr, "[fail baseline] %s/%s/%s: code=%d err=%v\n", c.template, c.language, b, code, err)
				continue
			}

			// concurrent
			r := runConcurrent(url, contentType, body, *concurrent, *concTotal, *timeout)
			sorted := append([]time.Duration(nil), r.lats...)
			sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
			var sum time.Duration
			for _, d := range sorted {
				sum += d
			}
			var avg time.Duration
			if len(sorted) > 0 {
				avg = sum / time.Duration(len(sorted))
			}
			throughput := 0.0
			if r.wall > 0 {
				throughput = float64(r.ok) / r.wall.Seconds()
			}
			// size_bytes 는 실제 전송한 body 기준 (reactpdf 는 JSON, 나머지는 HTML).
			// 입력 형식이 다르다는 것을 결과에서 추적 가능.
			fmt.Printf(rowFmt,
				c.template, c.language, b, len(body)/1024,
				baseline.Round(time.Millisecond),
				throughput,
				avg.Round(time.Millisecond),
				percentile(sorted, 0.50).Round(time.Millisecond),
				percentile(sorted, 0.95).Round(time.Millisecond),
				maxDur(sorted).Round(time.Millisecond),
				r.ok, r.fail,
			)

			w.Write([]string{
				c.template, c.language, b,
				strconv.Itoa(len(body)),
				strconv.FormatInt(baseline.Milliseconds(), 10),
				strconv.FormatFloat(throughput, 'f', 3, 64),
				strconv.FormatInt(avg.Milliseconds(), 10),
				strconv.FormatInt(percentile(sorted, 0.50).Milliseconds(), 10),
				strconv.FormatInt(percentile(sorted, 0.95).Milliseconds(), 10),
				strconv.FormatInt(maxDur(sorted).Milliseconds(), 10),
				strconv.Itoa(r.ok),
				strconv.Itoa(r.fail),
			})
			w.Flush()
			if *cooldown > 0 {
				time.Sleep(*cooldown)
			}
		}
	}
	fmt.Printf("\n총 소요: %s\n", time.Since(startAll).Round(time.Second))
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func setFromCSV(s string) map[string]bool {
	parts := splitCSV(s)
	if len(parts) == 0 {
		return nil
	}
	m := make(map[string]bool, len(parts))
	for _, p := range parts {
		m[p] = true
	}
	return m
}

func filterCombos(in []combo, tpl, lang map[string]bool) []combo {
	out := make([]combo, 0, len(in))
	for _, c := range in {
		if tpl != nil && !tpl[c.template] {
			continue
		}
		if lang != nil && !lang[c.language] {
			continue
		}
		out = append(out, c)
	}
	return out
}

type result struct {
	wall time.Duration
	lats []time.Duration
	ok   int
	fail int
}

func runConcurrent(url, contentType string, body []byte, concurrency, total int, timeout time.Duration) result {
	jobs := make(chan struct{}, total)
	for i := 0; i < total; i++ {
		jobs <- struct{}{}
	}
	close(jobs)

	var (
		mu   sync.Mutex
		lats = make([]time.Duration, 0, total)
		ok   int
		fail int
	)
	start := time.Now()
	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range jobs {
				t0 := time.Now()
				_, code, err := doOne(url, contentType, body, timeout)
				d := time.Since(t0)
				mu.Lock()
				lats = append(lats, d)
				if err == nil && code == 200 {
					ok++
				} else {
					fail++
				}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	return result{wall: time.Since(start), lats: lats, ok: ok, fail: fail}
}

func doOne(url, contentType string, body []byte, timeout time.Duration) (int, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return 0, 0, err
	}
	req.Header.Set("Content-Type", contentType)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()
	n, err := io.Copy(io.Discard, resp.Body)
	if err != nil {
		return int(n), resp.StatusCode, err
	}
	return int(n), resp.StatusCode, nil
}

func percentile(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(float64(len(sorted)-1) * p)
	return sorted[idx]
}

func maxDur(sorted []time.Duration) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	return sorted[len(sorted)-1]
}
