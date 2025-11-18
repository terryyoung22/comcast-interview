package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Caption interval
type Caption struct {
	Start float64
	End   float64
	Text  string
}

func main() {
	tStartStr := flag.String("t_start", "", "start time (seconds or HH:MM:SS or MM:SS)")
	tEndStr := flag.String("t_end", "", "end time (seconds or HH:MM:SS or MM:SS)")
	minPct := flag.Float64("min_coverage_pct", 80, "minimum coverage percent")
	endpoint := flag.String("endpoint", "", "language detection endpoint")
	flag.Parse()

	if *tStartStr == "" || *tEndStr == "" || *endpoint == "" || len(flag.Args()) != 1 {
		fmt.Fprintln(os.Stderr, "usage: program -t_start X -t_end Y -endpoint URL file.srt|vtt")
		os.Exit(2)
	}

	filePath := flag.Args()[0]

	start, err := parseTime(*tStartStr)
	exitOnErr("invalid t_start", err)
	end, err := parseTime(*tEndStr)
	exitOnErr("invalid t_end", err)
	if end <= start {
		fmt.Fprintln(os.Stderr, "t_end must be > t_start")
		os.Exit(2)
	}

	b, err := ioutil.ReadFile(filePath)
	exitOnErr("read file", err)
	ext := strings.ToLower(filepath.Ext(filePath))

	var caps []Caption
	switch ext {
	case ".srt":
		caps, err = parseSRT(string(b))
	case ".vtt":
		caps, err = parseVTT(string(b))
	default:
		// unsupported type
		os.Exit(1)
	}
	exitOnErr("parse captions", err)

	coverage := computeCoverage(caps, start, end)
	if coverage < *minPct {
		printJSON(map[string]interface{}{
			"type":        "caption_coverage",
			"required":    *minPct,
			"actual":      coverage,
			"t_start":     start,
			"t_end":       end,
			"explanation": "coverage too low",
		})
	}

	allText := joinText(caps)
	lang, err := detectLanguage(*endpoint, allText)
	exitOnErr("detect language", err)

	if lang != "en-US" {
		printJSON(map[string]interface{}{
			"type":        "incorrect_language",
			"expected":    "en-US",
			"detected":    lang,
			"explanation": "language mismatch",
		})
	}

	os.Exit(0)
}

/***********************
 Helper / parsing code
***********************/

func parseTime(s string) (float64, error) {
	// allow float seconds
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f, nil
	}
	parts := strings.Split(s, ":")
	if len(parts) == 3 {
		h, _ := strconv.ParseFloat(parts[0], 64)
		m, _ := strconv.ParseFloat(parts[1], 64)
		sec, _ := strconv.ParseFloat(parts[2], 64)
		return h*3600 + m*60 + sec, nil
	}
	if len(parts) == 2 {
		m, _ := strconv.ParseFloat(parts[0], 64)
		sec, _ := strconv.ParseFloat(parts[1], 64)
		return m*60 + sec, nil
	}
	return 0, fmt.Errorf("unrecognized time: %s", s)
}

// Simple SRT: split on double-newlines and find the "-->" line
func parseSRT(s string) ([]Caption, error) {
	blocks := splitBlocks(s)
	var out []Caption
	for _, b := range blocks {
		lines := nonEmptyLines(b)
		if len(lines) < 2 {
			continue
		}
		// find timestamp line
		var tsLine string
		for i := 0; i < len(lines); i++ {
			if strings.Contains(lines[i], "-->") {
				tsLine = lines[i]
				// text is rest
				text := strings.Join(lines[i+1:], "\n")
				start, end, err := parseRange(tsLine)
				if err != nil {
					return nil, err
				}
				out = append(out, Caption{Start: start, End: end, Text: text})
				break
			}
		}
	}
	return out, nil
}

// Simple VTT: ignore header, same approach
func parseVTT(s string) ([]Caption, error) {
	s = strings.Replace(s, "WEBVTT", "", 1)
	blocks := splitBlocks(s)
	var out []Caption
	for _, b := range blocks {
		lines := nonEmptyLines(b)
		if len(lines) == 0 {
			continue
		}
		// find timestamp line
		for i := 0; i < len(lines); i++ {
			if strings.Contains(lines[i], "-->") {
				start, end, err := parseRange(lines[i])
				if err != nil {
					return nil, err
				}
				text := strings.Join(lines[i+1:], "\n")
				out = append(out, Caption{Start: start, End: end, Text: text})
				break
			}
		}
	}
	return out, nil
}

func parseRange(line string) (float64, float64, error) {
	parts := strings.Split(line, "-->")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("bad range line: %s", line)
	}
	left := strings.TrimSpace(parts[0])
	right := strings.TrimSpace(parts[1])
	left = strings.Replace(left, ",", ".", 1)
	right = strings.Replace(right, ",", ".", 1)
	s1, err := parseTime(left)
	if err != nil {
		return 0, 0, err
	}
	s2, err := parseTime(right)
	if err != nil {
		return 0, 0, err
	}
	return s1, s2, nil
}

/*** small helpers ***/
func splitBlocks(s string) []string {
	// normalize newlines then split by blank line
	s = strings.ReplaceAll(s, "\r\n", "\n")
	parts := strings.Split(s, "\n\n")
	var out []string
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			out = append(out, p)
		}
	}
	return out
}

func nonEmptyLines(s string) []string {
	lines := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	var out []string
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			out = append(out, l)
		}
	}
	return out
}

func computeCoverage(caps []Caption, windowStart, windowEnd float64) float64 {
	var segs [][2]float64
	for _, c := range caps {
		if c.End <= windowStart || c.Start >= windowEnd {
			continue
		}
		s := max(c.Start, windowStart)
		e := min(c.End, windowEnd)
		if e > s {
			segs = append(segs, [2]float64{s, e})
		}
	}
	if len(segs) == 0 {
		return 0
	}
	sort.Slice(segs, func(i, j int) bool { return segs[i][0] < segs[j][0] })
	merged := make([][2]float64, 0, len(segs))
	curr := segs[0]
	for i := 1; i < len(segs); i++ {
		if segs[i][0] <= curr[1] {
			if segs[i][1] > curr[1] {
				curr[1] = segs[i][1]
			}
		} else {
			merged = append(merged, curr)
			curr = segs[i]
		}
	}
	merged = append(merged, curr)
	total := 0.0
	for _, m := range merged {
		total += m[1] - m[0]
	}
	window := windowEnd - windowStart
	if window <= 0 {
		return 0
	}
	return (total / window) * 100
}

func joinText(caps []Caption) string {
	var b strings.Builder
	for _, c := range caps {
		if strings.TrimSpace(c.Text) == "" {
			continue
		}
		b.WriteString(c.Text)
		b.WriteString("\n")
	}
	return b.String()
}

func detectLanguage(url, text string) (string, error) {
	resp, err := http.Post(url, "text/plain", strings.NewReader(text))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var out struct {
		Lang string `json:"lang"`
	}
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&out); err != nil {
		return "", err
	}
	return out.Lang, nil
}

func printJSON(v map[string]interface{}) {
	b, _ := json.Marshal(v)
	fmt.Println(string(b))
}

func exitOnErr(msg string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", msg, err)
		os.Exit(2)
	}
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
