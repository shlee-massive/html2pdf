package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"math"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ─────────────────────────────────────────────────────────────
// Data types
// ─────────────────────────────────────────────────────────────

type Party struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	TaxID   string `json:"tax_id,omitempty"`
	Contact string `json:"contact,omitempty"`
}

type Item struct {
	Description string  `json:"description"`
	SKU         string  `json:"sku,omitempty"`
	Quantity    int     `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
	Amount      float64 `json:"amount"`
}

type Invoice struct {
	Locale        string  `json:"locale"`
	Currency      string  `json:"currency"`
	InvoiceNumber string  `json:"invoice_number"`
	IssueDate     string  `json:"issue_date"`
	DueDate       string  `json:"due_date"`
	TaxRate       float64 `json:"tax_rate"`
	From          Party   `json:"from"`
	To            Party   `json:"to"`
	Items         []Item  `json:"items"`
	Notes         string  `json:"notes,omitempty"`

	// 계산된 필드 + 다국어 문자열
	Subtotal float64
	Tax      float64
	Total    float64
	Strings  Strings
}

type Strings struct {
	Title       string
	InvoiceNo   string
	IssueDate   string
	DueDate     string
	From        string
	To          string
	TaxID       string
	Description string
	Quantity    string
	UnitPrice   string
	Amount      string
	Subtotal    string
	Tax         string
	Total       string
	Page        string
	Stamp       string
}

var localeStrings = map[string]Strings{
	"ko": {
		Title: "청구서", InvoiceNo: "청구번호",
		IssueDate: "발행일", DueDate: "결제기한",
		From: "공급자", To: "공급받는자", TaxID: "사업자등록번호",
		Description: "품목", Quantity: "수량",
		UnitPrice: "단가", Amount: "금액",
		Subtotal: "공급가액", Tax: "부가세", Total: "합계",
		Page: "페이지", Stamp: "인감",
	},
	"ja": {
		Title: "請求書", InvoiceNo: "請求番号",
		IssueDate: "発行日", DueDate: "支払期限",
		From: "請求元", To: "請求先", TaxID: "登録番号",
		Description: "品目", Quantity: "数量",
		UnitPrice: "単価", Amount: "金額",
		Subtotal: "小計", Tax: "消費税", Total: "合計",
		Page: "ページ", Stamp: "印",
	},
	"en": {
		Title: "INVOICE", InvoiceNo: "Invoice No.",
		IssueDate: "Issue Date", DueDate: "Due Date",
		From: "From", To: "Bill To", TaxID: "Tax ID",
		Description: "Description", Quantity: "Qty",
		UnitPrice: "Unit Price", Amount: "Amount",
		Subtotal: "Subtotal", Tax: "Tax", Total: "Total",
		Page: "Page", Stamp: "Seal",
	},
}

// ─────────────────────────────────────────────────────────────
// Money formatting (stdlib only)
// ─────────────────────────────────────────────────────────────

func formatMoney(amount float64, currency string) string {
	var sym string
	var decimals int
	switch currency {
	case "KRW":
		sym, decimals = "₩", 0
	case "JPY":
		sym, decimals = "¥", 0
	case "USD":
		sym, decimals = "$", 2
	default:
		sym, decimals = currency+" ", 2
	}
	neg := amount < 0
	if neg {
		amount = -amount
	}
	out := sym + addThousandSep(amount, decimals)
	if neg {
		out = "-" + out
	}
	return out
}

func addThousandSep(n float64, decimals int) string {
	neg := n < 0
	if neg {
		n = -n
	}
	s := strconv.FormatFloat(math.Round(n*pow10(decimals))/pow10(decimals), 'f', decimals, 64)
	intPart, fracPart, _ := strings.Cut(s, ".")

	var b strings.Builder
	for i, c := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	out := b.String()
	if fracPart != "" {
		out += "." + fracPart
	}
	if neg {
		out = "-" + out
	}
	return out
}

func pow10(n int) float64 {
	p := 1.0
	for i := 0; i < n; i++ {
		p *= 10
	}
	return p
}

// ─────────────────────────────────────────────────────────────
// Rendering
// ─────────────────────────────────────────────────────────────

func loadInvoice(path string) (*Invoice, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var inv Invoice
	if err := json.Unmarshal(b, &inv); err != nil {
		return nil, err
	}
	str, ok := localeStrings[inv.Locale]
	if !ok {
		return nil, fmt.Errorf("unsupported locale: %s", inv.Locale)
	}
	inv.Strings = str

	// 합계 계산 (데이터에 amount가 있어도 재검증)
	var subtotal float64
	for _, it := range inv.Items {
		subtotal += it.Amount
	}
	inv.Subtotal = subtotal
	inv.Tax = math.Round(subtotal*inv.TaxRate) / 100
	inv.Total = inv.Subtotal + inv.Tax
	return &inv, nil
}

func renderHTML(tmplPath string, inv *Invoice) (string, error) {
	funcs := template.FuncMap{
		"inc":   func(i int) int { return i + 1 },
		"money": formatMoney,
	}
	t, err := template.New(filepath.Base(tmplPath)).Funcs(funcs).ParseFiles(tmplPath)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, inv); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// ─────────────────────────────────────────────────────────────
// Backends
// ─────────────────────────────────────────────────────────────

type Backend interface {
	Name() string
	Convert(ctx context.Context, html string) ([]byte, error)
}

// InvoiceBackend 는 HTML 이 아닌 구조화된 Invoice 데이터를 받는 백엔드용 옵셔널 인터페이스.
// react-pdf 처럼 자체 컴포넌트 트리로 PDF 를 만드는 엔진은 이 인터페이스를 구현한다.
// 비교 단위가 "HTML→PDF 변환 비용" → "같은 콘텐츠 end-to-end 생성 비용" 으로 한 단계 추상화된다.
// 측정 결과 해석 시 반드시 reports/ 에 명시할 것.
type InvoiceBackend interface {
	Backend
	ConvertInvoice(ctx context.Context, inv *Invoice) ([]byte, error)
}

// Gotenberg: POST /forms/chromium/convert/html with multipart files
type Gotenberg struct{ BaseURL string }

func (g Gotenberg) Name() string { return "gotenberg" }
func (g Gotenberg) Convert(ctx context.Context, html string) ([]byte, error) {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	fw, err := w.CreateFormFile("files", "index.html")
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(fw, strings.NewReader(html)); err != nil {
		return nil, err
	}
	// A4 + 인쇄 배경 + 페이지 사이즈
	_ = w.WriteField("paperWidth", "8.27")  // inch
	_ = w.WriteField("paperHeight", "11.7") // inch
	_ = w.WriteField("printBackground", "true")
	_ = w.WriteField("preferCssPageSize", "true")
	w.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		strings.TrimRight(g.BaseURL, "/")+"/forms/chromium/convert/html", &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	return doPDFRequest(req)
}

// WeasyPrint sidecar (this repo의 weasyprint/server.py)
type WeasyPrint struct{ BaseURL string }

func (w WeasyPrint) Name() string { return "weasyprint" }
func (w WeasyPrint) Convert(ctx context.Context, html string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		strings.TrimRight(w.BaseURL, "/")+"/pdf", strings.NewReader(html))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "text/html; charset=utf-8")
	return doPDFRequest(req)
}

// DocRaptor: REST API (test mode = 무료, "TEST" 워터마크)
type DocRaptor struct {
	APIKey string
	Test   bool
}

func (d DocRaptor) Name() string { return "docraptor" }
func (d DocRaptor) Convert(ctx context.Context, html string) ([]byte, error) {
	payload := map[string]any{
		"user_credentials": d.APIKey,
		"doc": map[string]any{
			"document_content": html,
			"type":             "pdf",
			"test":             d.Test,
			"prince_options": map[string]any{
				"media": "print",
			},
		},
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://docraptor.com/docs", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return doPDFRequest(req)
}

// ReactPdf sidecar (this repo의 react-pdf/server.mjs).
// HTML 을 받지 않고 Invoice JSON 을 받는 점이 다른 백엔드와 다르다.
// Backend.Convert(html) 호출은 그대로 받되, 내부에서 마지막 set 된 Invoice 를 재사용한다 — 다만
// 실제 호출 경로(batch loop)는 InvoiceBackend.ConvertInvoice 로 우선 라우팅되므로
// Convert 는 사실상 fallback (HTML 입력으로는 작동 불가).
type ReactPdf struct{ BaseURL string }

func (r ReactPdf) Name() string { return "reactpdf" }

func (r ReactPdf) Convert(ctx context.Context, html string) ([]byte, error) {
	return nil, fmt.Errorf("reactpdf 백엔드는 HTML 입력을 받지 않음 — ConvertInvoice 경로로 호출 필요")
}

func (r ReactPdf) ConvertInvoice(ctx context.Context, inv *Invoice) ([]byte, error) {
	body, err := json.Marshal(inv)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		strings.TrimRight(r.BaseURL, "/")+"/pdf", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return doPDFRequest(req)
}

func doPDFRequest(req *http.Request) ([]byte, error) {
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, truncate(string(buf), 500))
	}
	return buf, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "...(truncated)"
}

// ─────────────────────────────────────────────────────────────
// CLI
// ─────────────────────────────────────────────────────────────

func main() {
	var (
		serve         = flag.Bool("serve", false, "HTTP 서버 모드로 실행 (데모 페이지 제공)")
		addr          = flag.String("addr", ":8080", "HTTP 리스닝 주소 (-serve 모드일 때)")
		locale        = flag.String("locale", "all", "ko | ja | en | all")
		backend       = flag.String("backend", "all", "gotenberg | weasyprint | docraptor | all")
		tmplPath      = flag.String("template", "templates/invoice.html.tmpl", "template path")
		dataDir       = flag.String("data", "data", "directory containing {locale}.json")
		outDir        = flag.String("out", "output", "output directory")
		gotenbergURL  = flag.String("gotenberg-url", env("GOTENBERG_URL", "http://localhost:3000"), "Gotenberg base URL")
		weasyURL      = flag.String("weasyprint-url", env("WEASYPRINT_URL", "http://localhost:5001"), "WeasyPrint base URL")
		reactPdfURL   = flag.String("reactpdf-url", env("REACTPDF_URL", "http://localhost:5002"), "react-pdf sidecar base URL")
		docraptorKey  = flag.String("docraptor-key", env("DOCRAPTOR_API_KEY", "YOUR_API_KEY_HERE"), "DocRaptor API key (test mode은 무료, 워터마크 있음)")
		docraptorTest = flag.Bool("docraptor-test", envBool("DOCRAPTOR_TEST", true), "DocRaptor test mode (무료/워터마크)")
		dumpHTML      = flag.Bool("dump-html", false, "렌더링된 HTML도 함께 저장")
	)
	flag.Parse()

	locales := expand(*locale, []string{"ko", "ja", "en"})
	backends := expand(*backend, []string{"gotenberg", "weasyprint", "docraptor", "reactpdf"})

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		log.Fatalf("mkdir: %v", err)
	}

	backendMap := map[string]Backend{
		"gotenberg":  Gotenberg{BaseURL: *gotenbergURL},
		"weasyprint": WeasyPrint{BaseURL: *weasyURL},
		"docraptor":  DocRaptor{APIKey: *docraptorKey, Test: *docraptorTest},
		"reactpdf":   ReactPdf{BaseURL: *reactPdfURL},
	}

	if *serve {
		if err := runServer(serverConfig{
			Addr:     *addr,
			DataDir:  *dataDir,
			TmplPath: *tmplPath,
			Backends: backendMap,
		}); err != nil {
			log.Fatalf("server: %v", err)
		}
		return
	}

	ctx := context.Background()
	var fails int
	for _, loc := range locales {
		dataPath := filepath.Join(*dataDir, loc+".json")
		inv, err := loadInvoice(dataPath)
		if err != nil {
			log.Printf("[%s] load: %v", loc, err)
			fails++
			continue
		}
		html, err := renderHTML(*tmplPath, inv)
		if err != nil {
			log.Printf("[%s] render: %v", loc, err)
			fails++
			continue
		}
		if *dumpHTML {
			htmlPath := filepath.Join(*outDir, loc+".html")
			if err := os.WriteFile(htmlPath, []byte(html), 0o644); err != nil {
				log.Printf("[%s] dump html: %v", loc, err)
			} else {
				log.Printf("[%s] wrote %s", loc, htmlPath)
			}
		}

		for _, bk := range backends {
			b, ok := backendMap[bk]
			if !ok {
				log.Printf("unknown backend: %s", bk)
				continue
			}
			start := time.Now()
			var pdf []byte
			var err error
			// react-pdf 처럼 구조화된 입력을 받는 백엔드는 InvoiceBackend 경로로 라우팅.
			if ib, ok := b.(InvoiceBackend); ok {
				pdf, err = ib.ConvertInvoice(ctx, inv)
			} else {
				pdf, err = b.Convert(ctx, html)
			}
			elapsed := time.Since(start)
			if err != nil {
				log.Printf("[%s/%s] FAIL (%s): %v", loc, b.Name(), elapsed.Round(time.Millisecond), err)
				fails++
				continue
			}
			outPath := filepath.Join(*outDir, fmt.Sprintf("%s-%s.pdf", loc, b.Name()))
			if err := os.WriteFile(outPath, pdf, 0o644); err != nil {
				log.Printf("[%s/%s] write: %v", loc, b.Name(), err)
				fails++
				continue
			}
			log.Printf("[%s/%s] OK %s (%d KB)", loc, b.Name(), elapsed.Round(time.Millisecond), len(pdf)/1024)
		}
	}
	if fails > 0 {
		os.Exit(1)
	}
}

func expand(v string, all []string) []string {
	if v == "all" || v == "" {
		return all
	}
	out := []string{}
	for _, x := range strings.Split(v, ",") {
		x = strings.TrimSpace(x)
		if x != "" {
			out = append(out, x)
		}
	}
	return out
}

func env(k, def string) string {
	if v, ok := os.LookupEnv(k); ok && v != "" {
		return v
	}
	return def
}

func envBool(k string, def bool) bool {
	v, ok := os.LookupEnv(k)
	if !ok {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}
