package plaud

// apiResponse wraps all Plaud API responses.
type apiResponse[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

// File represents a recording in the file list.
type File struct {
	ID            string   `json:"id"`
	Filename      string   `json:"filename"`
	Duration      int64    `json:"duration"`   // milliseconds
	StartTime     int64    `json:"start_time"` // unix timestamp ms
	EndTime       int64    `json:"end_time"`   // unix timestamp ms
	IsTrash       bool     `json:"is_trash"`
	Scene         int      `json:"scene"`
	Filesize      int64    `json:"filesize"`
	EditFrom      string   `json:"edit_from"`
	IsTrans       bool     `json:"is_trans"`   // has transcript
	IsSummary     bool     `json:"is_summary"` // has AI summary
	FiletagIDList []string `json:"filetag_id_list"`
}

// Status returns a human-readable processing state for the file.
func (f File) Status() string {
	if f.IsTrans && f.IsSummary {
		return "Ready"
	}
	if f.IsTrans {
		return "Transcribed"
	}
	return "Processing"
}

// fileListData is the shape of the /file/simple/web response.
type fileListData struct {
	Status        int    `json:"status"`
	Msg           string `json:"msg"`
	DataFileTotal int    `json:"data_file_total"`
	DataFileList  []File `json:"data_file_list"`
}

// FileDetail represents the full detail response for a single file.
type FileDetail struct {
	FileID      string        `json:"file_id"`
	FileName    string        `json:"file_name"`
	Duration    int64         `json:"duration"`
	StartTime   int64         `json:"start_time"`
	IsTrash     bool          `json:"is_trash"`
	Scene       int           `json:"scene"`
	ContentList []ContentItem `json:"content_list"`
}

// ContentItem represents one content entry in a file detail.
// data_type values include:
//   - "transaction"     — transcript segments (pre-signed S3 link to gzipped JSON)
//   - "auto_sum_note"   — AI-generated summary
//   - "high_light"      — highlights
type ContentItem struct {
	DataType string `json:"data_type"`
	DataLink string `json:"data_link"`
	Language string `json:"language"`
}

// TranscriptResult is the parsed structure of the transcript JSON downloaded from S3.
type TranscriptResult struct {
	Segments []TranscriptSegment `json:"trans_result"`
}

// TranscriptSegment represents one segment of a transcript.
type TranscriptSegment struct {
	Content         string `json:"content"`
	Speaker         string `json:"speaker"`
	OriginalSpeaker string `json:"original_speaker"`
	StartTime       int64  `json:"start_time"` // milliseconds
	EndTime         int64  `json:"end_time"`   // milliseconds
}

// tempURLResponse is the response from /file/temp-url/{id}.
type tempURLResponse struct {
	TempURL string `json:"temp_url"`
}

// ListFilesOptions configures the ListFiles call.
type ListFilesOptions struct {
	Skip         int
	Limit        int
	IncludeTrash bool
	SortBy       string // default: "start_time"
	Descending   bool   // default: true
}
