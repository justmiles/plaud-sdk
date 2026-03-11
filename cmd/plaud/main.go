package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"text/tabwriter"
	"time"

	plaud "github.com/justmiles/plaud-sdk"
)

func usage() {
	fmt.Fprintf(os.Stderr, `Plaud CLI - interact with your Plaud recordings

Usage:
  plaud list                        List all recordings
  plaud text <fileID>               Print transcript as plain text
  plaud segments <fileID>           Print transcript segments as JSON-like output
  plaud audio <fileID> <output>     Download audio to file
  plaud debug                       Dump raw API response for debugging
  plaud debug-transcript <fileID>   Dump raw transcript JSON for debugging

Environment:
  PLAUD_TOKEN    Bearer token (required)
                 Get it from https://web.plaud.ai DevTools console:
                 localStorage.getItem('tokenstr')

  PLAUD_BASE_URL Override API base URL (optional)
`)
	os.Exit(1)
}

func main() {
	if len(os.Args) < 2 {
		usage()
	}

	token := os.Getenv("PLAUD_TOKEN")
	if token == "" {
		fmt.Fprintln(os.Stderr, "Error: PLAUD_TOKEN environment variable is required.")
		fmt.Fprintln(os.Stderr, "Get your token from https://web.plaud.ai DevTools console:")
		fmt.Fprintln(os.Stderr, "  localStorage.getItem('tokenstr')")
		os.Exit(1)
	}

	var opts []plaud.Option
	if base := os.Getenv("PLAUD_BASE_URL"); base != "" {
		opts = append(opts, plaud.WithBaseURL(base))
	}

	client := plaud.New(token, opts...)
	ctx := context.Background()

	switch os.Args[1] {
	case "list":
		cmdList(ctx, client)
	case "text":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: plaud text <fileID>")
			os.Exit(1)
		}
		cmdText(ctx, client, os.Args[2])
	case "segments":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: plaud segments <fileID>")
			os.Exit(1)
		}
		cmdSegments(ctx, client, os.Args[2])
	case "audio":
		if len(os.Args) < 4 {
			fmt.Fprintln(os.Stderr, "Usage: plaud audio <fileID> <output.mp3>")
			os.Exit(1)
		}
		cmdAudio(ctx, client, os.Args[2], os.Args[3])
	case "debug":
		cmdDebug(ctx, client)
	case "debug-transcript":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: plaud debug-transcript <fileID>")
			os.Exit(1)
		}
		cmdDebugTranscript(ctx, client, os.Args[2])
	default:
		usage()
	}
}

func cmdList(ctx context.Context, client *plaud.Client) {
	files, err := client.ListFiles(ctx, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintf(w, "ID\tNAME\tDURATION\tSTATUS\tDATE\n")
	for _, f := range files {
		dur := formatDuration(f.Duration)
		date := time.UnixMilli(f.StartTime).Format("2006-01-02 15:04")
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", f.ID, f.Filename, dur, f.Status(), date)
	}
	w.Flush()

	fmt.Fprintf(os.Stderr, "\n%d files\n", len(files))
}

func cmdText(ctx context.Context, client *plaud.Client, fileID string) {
	text, err := client.GetTranscriptText(ctx, fileID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(text)
}

func cmdSegments(ctx context.Context, client *plaud.Client, fileID string) {
	segments, err := client.GetTranscriptSegments(ctx, fileID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	for _, seg := range segments {
		start := formatDuration(seg.StartTime)
		end := formatDuration(seg.EndTime)
		speaker := seg.Speaker
		if speaker == "" {
			speaker = "Unknown"
		}
		fmt.Printf("[%s - %s] %s: %s\n", start, end, speaker, seg.Content)
	}
}

func cmdAudio(ctx context.Context, client *plaud.Client, fileID string, output string) {
	f, err := os.Create(output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	fmt.Fprintf(os.Stderr, "Downloading audio for %s...\n", fileID)
	if err := client.GetAudio(ctx, fileID, f); err != nil {
		os.Remove(output)
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	info, _ := f.Stat()
	fmt.Fprintf(os.Stderr, "Saved to %s (%d bytes)\n", output, info.Size())
}

func cmdDebug(ctx context.Context, client *plaud.Client) {
	resp, err := client.Do(ctx, "GET", "/file/simple/web?skip=0&limit=5&is_trash=2&sort_by=start_time&is_desc=true", nil, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	fmt.Fprintf(os.Stderr, "Status: %d\n", resp.StatusCode)
	fmt.Fprintf(os.Stderr, "Content-Type: %s\n", resp.Header.Get("Content-Type"))
	b, _ := io.ReadAll(resp.Body)
	fmt.Println(string(b))
}

func cmdDebugTranscript(ctx context.Context, client *plaud.Client, fileID string) {
	detail, err := client.GetFileDetail(ctx, fileID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting detail: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "Content items:\n")
	for _, item := range detail.ContentList {
		fmt.Fprintf(os.Stderr, "  type=%s link=%s\n", item.DataType, item.DataLink[:min(80, len(item.DataLink))])
	}

	// Download raw transcript
	for _, item := range detail.ContentList {
		if item.DataType == "transaction" {
			req, _ := http.NewRequestWithContext(ctx, "GET", item.DataLink, nil)
			httpResp, err := http.DefaultClient.Do(req)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error downloading: %v\n", err)
				os.Exit(1)
			}
			defer httpResp.Body.Close()
			raw, _ := io.ReadAll(httpResp.Body)

			// Try gzip
			if gz, err := gzip.NewReader(bytes.NewReader(raw)); err == nil {
				decompressed, _ := io.ReadAll(gz)
				raw = decompressed
			}

			if len(raw) > 2000 {
				raw = raw[:2000]
			}
			fmt.Println(string(raw))
			return
		}
	}
	fmt.Fprintln(os.Stderr, "No transcript content item found")
}

func formatDuration(ms int64) string {
	s := ms / 1000
	m := s / 60
	s = s % 60
	h := m / 60
	m = m % 60
	if h > 0 {
		return fmt.Sprintf("%dh%dm%ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm%ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}
