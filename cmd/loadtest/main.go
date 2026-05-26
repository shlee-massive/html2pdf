// loadtest: PDF 변환 엔드포인트 동시 요청 부하 테스트
//
// 사용 예:
//   go run ./cmd/loadtest -backend gotenberg -concurrency 4 -total 20
//   go run ./cmd/loadtest -backend gotenberg -sweep
//   go run ./cmd/loadtest -backend weasyprint -sample output/ja.html -sweep
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	var (
		baseURL     = flag.String("url", "http://localhost:8080", "서버 base URL")
		backend     = flag.String("backend", "gotenberg", "백엔드 이름 (gotenberg|weasyprint)")
		samplePath  = flag.String("sample", "output/en.html", "POST 본문으로 보낼 HTML 파일")
		concurrency = flag.Int("concurrency", 4, "동시 요청 수")
		total       = flag.Int("total", 20, "총 요청 수")
		timeout     = flag.Duration("timeout", 120*time.Second, "요청 타임아웃")
		sweep       = flag.Bool("sweep", false, "동시성 1,2,4,8,16 자동 스윕")
		sweepTotal  = flag.Int("sweep-total", 16, "스윕 시 레벨당 요청 수")
		warmup      = flag.Bool("warmup", true, "측정 전 1회 워밍업 요청")
	)
	flag.Parse()

	html, err := os.ReadFile(*samplePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "샘플 HTML 읽기 실패 (%s): %v\n", *samplePath, err)
		fmt.Fprintf(os.Stderr, "힌트: 먼저 `make run` 등으로 output/*.html 을 생성하세요.\n")
		os.Exit(1)
	}

	url := *baseURL + "/api/convert?backend=" + *backend
	fmt.Printf("Target : %s\n", url)
	fmt.Printf("Sample : %s (%d KB)\n", *samplePath, len(html)/1024)
	fmt.Printf("Timeout: %s\n", *timeout)

	if !ping(*baseURL) {
		fmt.Fprintf(os.Stderr, "\n[warn] %s/api/health 응답 없음. 서버가 켜져 있는지 확인하세요 (make serve).\n", *baseURL)
	}

	if *warmup {
		fmt.Printf("\n[warmup] 1회 요청 중...\n")
		_, code, err := doOne(url, html, *timeout)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[warmup] 실패: %v (status=%d)\n", err, code)
			os.Exit(1)
		}
		fmt.Printf("[warmup] OK (status=%d)\n", code)
	}

	if *sweep {
		fmt.Printf("\n=== 동시성 스윕 (각 레벨당 %d 요청) ===\n", *sweepTotal)
		printHeader()
		for _, c := range []int{1, 2, 4, 8, 16} {
			r := run(url, html, c, *sweepTotal, *timeout)
			printRow(c, r)
		}
		return
	}

	fmt.Printf("\n=== 부하 테스트: concurrency=%d total=%d ===\n", *concurrency, *total)
	printHeader()
	r := run(url, html, *concurrency, *total, *timeout)
	printRow(*concurrency, r)
}

type result struct {
	wall    time.Duration
	lats    []time.Duration
	ok      int
	fail    int
	bytes   int64
	codes   map[int]int
	firstEr string
}

func run(url string, body []byte, concurrency, total int, timeout time.Duration) result {
	jobs := make(chan struct{}, total)
	for i := 0; i < total; i++ {
		jobs <- struct{}{}
	}
	close(jobs)

	lats := make([]time.Duration, 0, total)
	var (
		mu       sync.Mutex
		okCnt    int64
		failCnt  int64
		bytesSum int64
		codes    = map[int]int{}
		firstEr  string
	)

	start := time.Now()
	var wg sync.WaitGroup
	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range jobs {
				t0 := time.Now()
				n, code, err := doOne(url, body, timeout)
				dur := time.Since(t0)
				mu.Lock()
				lats = append(lats, dur)
				codes[code]++
				if err == nil && code == 200 {
					atomic.AddInt64(&okCnt, 1)
					atomic.AddInt64(&bytesSum, int64(n))
				} else {
					atomic.AddInt64(&failCnt, 1)
					if firstEr == "" {
						if err != nil {
							firstEr = err.Error()
						} else {
							firstEr = fmt.Sprintf("status %d", code)
						}
					}
				}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	wall := time.Since(start)

	return result{
		wall:    wall,
		lats:    lats,
		ok:      int(okCnt),
		fail:    int(failCnt),
		bytes:   bytesSum,
		codes:   codes,
		firstEr: firstEr,
	}
}

func doOne(url string, body []byte, timeout time.Duration) (int, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return 0, 0, err
	}
	req.Header.Set("Content-Type", "text/html; charset=utf-8")
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

func ping(base string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, base+"/api/health", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

func percentile(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(float64(len(sorted)-1) * p)
	return sorted[idx]
}

func printHeader() {
	fmt.Printf("%4s | %5s %5s | %8s | %7s | %8s %8s %8s %8s %8s | %s\n",
		"conc", "ok", "fail", "wall", "req/s", "avg", "p50", "p95", "p99", "max", "codes")
	fmt.Println("-----+-------------+----------+---------+-------------------------------------------------+------------")
}

func printRow(conc int, r result) {
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
	fmt.Printf("%4d | %5d %5d | %8s | %7.2f | %8s %8s %8s %8s %8s | %v\n",
		conc, r.ok, r.fail,
		r.wall.Round(time.Millisecond),
		throughput,
		avg.Round(time.Millisecond),
		percentile(sorted, 0.50).Round(time.Millisecond),
		percentile(sorted, 0.95).Round(time.Millisecond),
		percentile(sorted, 0.99).Round(time.Millisecond),
		sorted[len(sorted)-1].Round(time.Millisecond),
		r.codes,
	)
	if r.firstEr != "" {
		fmt.Printf("       first error: %s\n", r.firstEr)
	}
}
