package plaud

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// GetTranscriptText returns the full transcript for a file as formatted text.
// Each segment is formatted as:
//
//	HH:MM:SS Speaker Name
//	Content text
func (c *Client) GetTranscriptText(ctx context.Context, fileID string) (string, error) {
	segments, err := c.GetTranscriptSegments(ctx, fileID)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	for i, seg := range segments {
		if i > 0 {
			b.WriteString("\n")
		}
		// Format timestamp as HH:MM:SS
		ts := seg.StartTime / 1000
		h := ts / 3600
		m := (ts % 3600) / 60
		s := ts % 60
		fmt.Fprintf(&b, "%02d:%02d:%02d %s\n", h, m, s, seg.Speaker)
		b.WriteString(seg.Content)
		b.WriteString("\n")
	}
	return b.String(), nil
}

// GetTranscriptSegments returns the parsed transcript segments for a file.
func (c *Client) GetTranscriptSegments(ctx context.Context, fileID string) ([]TranscriptSegment, error) {
	detail, err := c.GetFileDetail(ctx, fileID)
	if err != nil {
		return nil, err
	}

	// Find the transcript content item
	var transcriptURL string
	for _, item := range detail.ContentList {
		if item.DataType == "transaction" {
			transcriptURL = item.DataLink
			break
		}
	}
	if transcriptURL == "" {
		return nil, fmt.Errorf("plaud: no transcript found for file %s", fileID)
	}

	// Download the transcript data from S3
	req, err := http.NewRequestWithContext(ctx, "GET", transcriptURL, nil)
	if err != nil {
		return nil, fmt.Errorf("plaud: creating S3 request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("plaud: downloading transcript: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("plaud: S3 error %d: %s", resp.StatusCode, string(b))
	}

	// Read full body into buffer so we can try gzip, then fall back to raw JSON
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("plaud: reading transcript response: %w", err)
	}

	var reader io.Reader
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		// Not gzipped — treat as raw JSON
		reader = bytes.NewReader(data)
	} else {
		defer gz.Close()
		reader = gz
	}

	var segments []TranscriptSegment
	if err := json.NewDecoder(reader).Decode(&segments); err != nil {
		return nil, fmt.Errorf("plaud: decoding transcript JSON: %w", err)
	}

	return segments, nil
}

// GetAudio downloads the audio (MP3) for a file and writes it to w.
func (c *Client) GetAudio(ctx context.Context, fileID string, w io.Writer) error {
	// Get the pre-signed temp URL
	var resp apiResponse[tempURLResponse]
	if err := c.doJSON(ctx, "GET", fmt.Sprintf("/file/temp-url/%s", fileID), nil, nil, &resp); err != nil {
		return err
	}

	if resp.Data.TempURL == "" {
		return fmt.Errorf("plaud: no audio URL returned for file %s", fileID)
	}

	// Download the audio from S3
	req, err := http.NewRequestWithContext(ctx, "GET", resp.Data.TempURL, nil)
	if err != nil {
		return fmt.Errorf("plaud: creating audio request: %w", err)
	}

	audioResp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("plaud: downloading audio: %w", err)
	}
	defer audioResp.Body.Close()

	if audioResp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(audioResp.Body)
		return fmt.Errorf("plaud: S3 audio error %d: %s", audioResp.StatusCode, string(b))
	}

	if _, err := io.Copy(w, audioResp.Body); err != nil {
		return fmt.Errorf("plaud: writing audio: %w", err)
	}

	return nil
}
