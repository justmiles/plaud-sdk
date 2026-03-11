package plaud

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// ListFiles returns all recordings in the account.
// Pass nil for opts to use defaults.
func (c *Client) ListFiles(ctx context.Context, opts *ListFilesOptions) ([]File, error) {
	if opts == nil {
		opts = &ListFilesOptions{}
	}

	// Apply defaults
	limit := opts.Limit
	if limit <= 0 {
		limit = 99999
	}
	sortBy := opts.SortBy
	if sortBy == "" {
		sortBy = "start_time"
	}
	// Default to descending (newest first)
	isDesc := true
	if opts.SortBy != "" {
		isDesc = opts.Descending
	}

	// is_trash: 2 = all (matches web client default), 0 = non-trash only, 1 = trash only
	isTrash := 2
	if !opts.IncludeTrash {
		isTrash = 2 // still use 2; the web client always uses 2
	}

	q := url.Values{}
	q.Set("skip", strconv.Itoa(opts.Skip))
	q.Set("limit", strconv.Itoa(limit))
	q.Set("is_trash", strconv.Itoa(isTrash))
	q.Set("sort_by", sortBy)
	q.Set("is_desc", strconv.FormatBool(isDesc))

	// The list endpoint returns {data_file_list: [...]} directly (no wrapper)
	var resp fileListData
	if err := c.doJSON(ctx, "GET", "/file/simple/web", q, nil, &resp); err != nil {
		return nil, err
	}

	files := resp.DataFileList
	if !opts.IncludeTrash {
		filtered := files[:0]
		for _, f := range files {
			if !f.IsTrash {
				filtered = append(filtered, f)
			}
		}
		files = filtered
	}

	return files, nil
}

// GetFileDetail returns full metadata for a single file, including content links.
func (c *Client) GetFileDetail(ctx context.Context, fileID string) (*FileDetail, error) {
	var resp apiResponse[FileDetail]
	if err := c.doJSON(ctx, "GET", fmt.Sprintf("/file/detail/%s", fileID), nil, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}
