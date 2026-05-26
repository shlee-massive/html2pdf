package main

import (
	"strings"
	"testing"
)

func TestRenderAllLocales(t *testing.T) {
	cases := []struct {
		path     string
		mustHave []string
	}{
		{"data/ko.json", []string{"청구서", "₩", "사업자등록번호"}},
		{"data/ja.json", []string{"請求書", "¥", "髙﨑𠮷田商事"}},
		{"data/en.json", []string{"INVOICE", "$", "Bill To"}},
	}
	for _, c := range cases {
		t.Run(c.path, func(t *testing.T) {
			inv, err := loadInvoice(c.path)
			if err != nil {
				t.Fatalf("load: %v", err)
			}
			html, err := renderHTML("templates/invoice.html.tmpl", inv)
			if err != nil {
				t.Fatalf("render: %v", err)
			}
			for _, s := range c.mustHave {
				if !strings.Contains(html, s) {
					t.Errorf("missing %q in rendered output", s)
				}
			}
			if inv.Subtotal <= 0 || inv.Total <= inv.Subtotal {
				t.Errorf("invalid totals: subtotal=%f total=%f", inv.Subtotal, inv.Total)
			}
		})
	}
}

func TestFormatMoney(t *testing.T) {
	tests := []struct {
		amount   float64
		currency string
		want     string
	}{
		{1234567, "KRW", "₩1,234,567"},
		{1234567, "JPY", "¥1,234,567"},
		{1234.56, "USD", "$1,234.56"},
		{0, "JPY", "¥0"},
		{-500, "USD", "-$500.00"},
	}
	for _, tt := range tests {
		got := formatMoney(tt.amount, tt.currency)
		if got != tt.want {
			t.Errorf("formatMoney(%v,%v) = %q, want %q", tt.amount, tt.currency, got, tt.want)
		}
	}
}
