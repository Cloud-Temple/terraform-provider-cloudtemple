package client

import (
	"context"
	"net/url"
)

type BucketFilesClient struct {
	c *Client
}

func (c *ObjectStorage) BucketFiles() *BucketFilesClient {
	return &BucketFilesClient{c.c}
}

type BucketFile struct {
	Key          string
	LastModified string
	Size         int64
	Tags         []struct {
		Key   string
		Value string
	}
	Versions []BucketFileVersion
}

type BucketFileVersion struct {
	VersionID      string
	IsLatest       bool
	LastModified   string
	Size           int64
	IsDeleteMarker bool
}

type BucketFilesFilter struct {
	FolderPath string `filter:"folderPath"`
}

func (c *BucketFilesClient) List(ctx context.Context, bucketName string, filter *BucketFilesFilter) ([]*BucketFile, error) {
	r := c.c.newRequest("GET", "/object-storage/v1/buckets/"+url.PathEscape(bucketName)+"/files")
	r.addFilter(filter)
	resp, err := c.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*BucketFile
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}
