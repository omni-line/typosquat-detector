// Command update-corpus refreshes embedded popular-package snapshots.
//
// Each ecosystem is one entry in sources. Adding a registry means adding a
// source with a fetch function; the output file name (<name>.json.gz) must
// match ecosystem.Ecosystem.Name.
package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const (
	pypiURL = "https://hugovk.github.io/top-pypi-packages/top-pypi-packages-30-days.min.json"
	npmURL  = "https://github.com/LeoDog896/npm-rank/releases/download/latest/raw.json"

	limit        = 10000
	maxBodyBytes = 256 << 20
	userAgent    = "typosquat-detector-update-corpus (+https://github.com/omni-line/typosquat-detector)"
)

// minNames guards against committing a truncated or empty upstream response.
const minNames = 1000

type source struct {
	name  string
	url   string
	fetch func(ctx context.Context, c *http.Client) ([]string, error)
	// fallback, if set, is used when fetch fails.
	fallback func() []string
}

var sources = []source{
	{name: "npm", url: npmURL, fetch: fetchNPM, fallback: npmSeed},
	{name: "pypi", url: pypiURL, fetch: fetchPyPI},
}

type ecoMeta struct {
	Count  int    `json:"count"`
	Source string `json:"source"`
}

type meta struct {
	Ecosystems  map[string]ecoMeta `json:"ecosystems"`
	GeneratedAt string             `json:"generated_at"`
}

func main() {
	outDir := flag.String("out", "internal/corpus", "output directory")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	err := run(ctx, *outDir)
	cancel()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, outDir string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	client := &http.Client{Timeout: 2 * time.Minute}
	m := meta{Ecosystems: map[string]ecoMeta{}, GeneratedAt: time.Now().UTC().Format(time.RFC3339)}
	for _, src := range sources {
		names, err := src.fetch(ctx, client)
		origin := src.url
		if err == nil && len(names) < minNames {
			err = fmt.Errorf("only %d names returned (want >= %d)", len(names), minNames)
		}
		if err != nil {
			if src.fallback == nil {
				return fmt.Errorf("%s: %w", src.name, err)
			}
			fmt.Fprintf(os.Stderr, "warning: %s fetch failed (%v); using seed list\n", src.name, err)
			names = src.fallback()
			origin = "seeded top " + src.name + " packages (scripts/update-corpus fallback)"
		}
		sort.Strings(names)
		if err := writeGzipJSON(filepath.Join(outDir, src.name+".json.gz"), names); err != nil {
			return err
		}
		m.Ecosystems[src.name] = ecoMeta{Count: len(names), Source: origin}
	}
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outDir, "meta.json"), append(raw, '\n'), 0o644)
}

func getJSON(ctx context.Context, c *http.Client, url string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: status %d", url, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes+1))
	if err != nil {
		return err
	}
	if len(body) > maxBodyBytes {
		return errors.New("response exceeds size limit")
	}
	return json.Unmarshal(body, v)
}

func fetchPyPI(ctx context.Context, c *http.Client) ([]string, error) {
	var payload struct {
		Rows []struct {
			Project string `json:"project"`
		} `json:"rows"`
	}
	if err := getJSON(ctx, c, pypiURL, &payload); err != nil {
		return nil, err
	}
	names := make([]string, 0, len(payload.Rows))
	for _, row := range payload.Rows {
		names = append(names, row.Project)
	}
	return topUnique(names), nil
}

func fetchNPM(ctx context.Context, c *http.Client) ([]string, error) {
	var rows []struct {
		Name string `json:"name"`
	}
	if err := getJSON(ctx, c, npmURL, &rows); err != nil {
		return nil, err
	}
	names := make([]string, 0, len(rows))
	for _, row := range rows {
		names = append(names, row.Name)
	}
	names = topUnique(names)
	// Ensure classic typosquat targets exist even if the rank dump drifts.
	return appendUnique(names, "react-dom", "cross-env", "lodash", "express", "axios", "typescript"), nil
}

// topUnique keeps the first limit distinct non-empty names, preserving the
// upstream popularity order.
func topUnique(names []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, limit)
	for _, n := range names {
		if n == "" {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
		if len(out) == limit {
			break
		}
	}
	return out
}

func appendUnique(names []string, extra ...string) []string {
	seen := make(map[string]struct{}, len(names))
	for _, n := range names {
		seen[n] = struct{}{}
	}
	for _, n := range extra {
		if _, ok := seen[n]; !ok {
			names = append(names, n)
			seen[n] = struct{}{}
		}
	}
	return names
}

func writeGzipJSON(path string, names []string) error {
	raw, err := json.Marshal(names)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(raw); err != nil {
		return err
	}
	if err := zw.Close(); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}
