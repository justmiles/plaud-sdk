# plaud-sdk

Go client SDK for the unofficial [Plaud](https://plaud.ai) API. List recordings, export transcripts as text, and download audio.

> **⚠️ Unofficial**: This SDK is not affiliated with Plaud. The API is reverse-engineered and may change without notice.

## Install

### Download Pre-built Binary

Download the latest release for your platform from [GitHub Releases](https://github.com/justmiles/plaud-sdk/releases).

### Build from Source

```bash
# Install dependencies via devbox
devbox shell

# Build
devbox run build

# Or install to GOPATH
go install ./cmd/plaud
```

## Get Your Token

1. Open [web.plaud.ai](https://web.plaud.ai) in Chrome  
2. Log in to your account  
3. Open DevTools (F12) → Console  
4. Run: `localStorage.getItem('tokenstr')`  
5. Copy the token value

```bash
export PLAUD_TOKEN="bearer eyJ..."
```

## CLI Usage

```bash
# List all recordings
plaud list

# Export transcript as plain text
plaud text <fileID>

# Export transcript with timestamps and speakers
plaud segments <fileID>

# Download audio as MP3
plaud audio <fileID> output.mp3
```

## Go SDK Usage

```go
package main

import (
    "context"
    "fmt"
    "os"

    plaud "github.com/justmiles/plaud-sdk"
)

func main() {
    client := plaud.New(os.Getenv("PLAUD_TOKEN"))
    ctx := context.Background()

    // List recordings
    files, _ := client.ListFiles(ctx, nil)
    for _, f := range files {
        fmt.Printf("%s  %s\n", f.ID, f.Filename)
    }

    // Get transcript as plain text
    text, _ := client.GetTranscriptText(ctx, files[0].ID)
    fmt.Println(text)

    // Get structured transcript segments
    segments, _ := client.GetTranscriptSegments(ctx, files[0].ID)
    for _, seg := range segments {
        fmt.Printf("[%s] %s\n", seg.Speaker, seg.Text)
    }

    // Download audio
    f, _ := os.Create("recording.mp3")
    defer f.Close()
    client.GetAudio(ctx, files[0].ID, f)
}
```

## API Reference

| Method | Description |
|---|---|
| `New(token, ...Option)` | Create client |
| `ListFiles(ctx, opts)` | List all recordings |
| `GetFileDetail(ctx, fileID)` | Get full file metadata |
| `GetTranscriptText(ctx, fileID)` | Export transcript as plain text |
| `GetTranscriptSegments(ctx, fileID)` | Get structured transcript segments |
| `GetAudio(ctx, fileID, writer)` | Download audio MP3 |

### Options

| Option | Description |
|---|---|
| `WithBaseURL(url)` | Override API base URL (default: `https://api.plaud.ai`) |
| `WithHTTPClient(client)` | Use custom `*http.Client` |
