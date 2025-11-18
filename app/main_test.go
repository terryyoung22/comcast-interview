package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseSRT(t *testing.T) {
	srt := `1
00:00:01,000 --> 00:00:03,000
Hello

2
00:00:05,000 --> 00:00:06,500
World
`
	caps, err := parseSRT(srt)
	if err != nil {
		t.Fatalf("parseSRT error: %v", err)
	}
	if len(caps) != 2 {
		t.Fatalf("expected 2 captions, got %d", len(caps))
	}
	if caps[0].Start != 1 || caps[0].End != 3 {
		t.Fatalf("unexpected times: %+v", caps[0])
	}
}

func TestParseVTT(t *testing.T) {
	vtt := `WEBVTT

00:00:00.000 --> 00:00:02.000
Hi there

00:00:03.000 --> 00:00:04.500
Bye
`
	caps, err := parseVTT(vtt)
	if err != nil {
		t.Fatalf("parseVTT error: %v", err)
	}
	if len(caps) != 2 {
		t.Fatalf("expected 2 captions, got %d", len(caps))
	}
}

func TestComputeCoverage(t *testing.T) {
	caps := []Caption{
		{Start: 0, End: 10},
		{Start: 8, End: 20},
	}
	pct := computeCoverage(caps, 0, 20)
	if pct != 100 {
		t.Fatalf("expected 100 got %v", pct)
	}
	pct2 := computeCoverage(caps, 5, 15) // covered 5..15 => 10 sec out of 10 => 100
	if pct2 != 100 {
		t.Fatalf("expected 100 got %v", pct2)
	}
	pct3 := computeCoverage(caps, 15, 30) // covered none -> 0
	if pct3 != 0 {
		t.Fatalf("expected 0 got %v", pct3)
	}
}

func TestDetectLanguage(t *testing.T) {
	// create a small httptest server returning JSON {"lang":"en-US"}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != "POST" {
			t.Fatalf("expected POST")
		}
		// read body just to ensure it's provided
		body, _ := io.ReadAll(r.Body)
		_ = body // ignore content, just ensuring body is read
		w.Write([]byte(`{"lang":"en-US"}`))
	}))
	defer ts.Close()

	lang, err := detectLanguage(ts.URL, "hello")
	if err != nil {
		t.Fatalf("detectLanguage error: %v", err)
	}
	if lang != "en-US" {
		t.Fatalf("expected en-US got %s", lang)
	}
}
