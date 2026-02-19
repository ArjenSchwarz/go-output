package output

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// TestS3WriterCSVAppend_NewlineHandling tests the CSV append functionality
// to ensure proper newline handling between existing and new data.
// This is a regression test for the bug where CSV rows could be merged
// without proper newline separation.
func TestS3WriterCSVAppend_NewlineHandling(t *testing.T) {
	tests := map[string]struct {
		existingCSV string
		newCSV      string
		wantResult  string
	}{
		"existing CSV without trailing newline": {
			existingCSV: "header1,header2\nrow1,data1",
			newCSV:      "header1,header2\nrow2,data2\n",
			wantResult:  "header1,header2\nrow1,data1\nrow2,data2\n",
		},
		"existing CSV with trailing newline": {
			existingCSV: "header1,header2\nrow1,data1\n",
			newCSV:      "header1,header2\nrow2,data2\n",
			wantResult:  "header1,header2\nrow1,data1\nrow2,data2\n",
		},
		"empty existing CSV": {
			existingCSV: "",
			newCSV:      "header1,header2\nrow1,data1\n",
			wantResult:  "header1,header2\nrow1,data1\n", // When existing is empty, full CSV is written
		},
		"new CSV with only header": {
			existingCSV: "header1,header2\nrow1,data1\n",
			newCSV:      "header1,header2\n",
			wantResult:  "header1,header2\nrow1,data1\n",
		},
		"multiple rows in new CSV": {
			existingCSV: "header1,header2\nrow1,data1",
			newCSV:      "header1,header2\nrow2,data2\nrow3,data3\n",
			wantResult:  "header1,header2\nrow1,data1\nrow2,data2\nrow3,data3\n",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()

			etag := "test-etag"
			contentLength := int64(len(tc.existingCSV))

			client := &mockS3Client{
				getObjectFunc: func(ctx context.Context, input *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
					if tc.existingCSV == "" {
						// Simulate object not found for empty existing CSV
						return nil, &types.NoSuchKey{
							Message: aws.String("The specified key does not exist."),
						}
					}
					return &s3.GetObjectOutput{
						Body:          io.NopCloser(strings.NewReader(tc.existingCSV)),
						ETag:          &etag,
						ContentLength: &contentLength,
					}, nil
				},
			}

			sw := NewS3WriterWithOptions(client, "test-bucket", "data.{ext}", WithS3AppendMode())

			err := sw.Write(ctx, FormatCSV, []byte(tc.newCSV))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify PutObject was called with properly combined data
			calls := client.getCalls()
			if len(calls) != 1 {
				t.Fatalf("expected 1 PutObject call, got %d", len(calls))
			}

			if calls[0].Body != tc.wantResult {
				t.Errorf("combined CSV = %q, want %q", calls[0].Body, tc.wantResult)
			}
		})
	}
}

// TestCombineCSVData_DirectUnit tests the combineCSVData function directly
// to ensure proper newline handling at the unit level.
func TestCombineCSVData_DirectUnit(t *testing.T) {
	tests := map[string]struct {
		existing []byte
		new      []byte
		want     []byte
	}{
		"existing without newline": {
			existing: []byte("header1,header2\nrow1,data1"),
			new:      []byte("header1,header2\nrow2,data2\n"),
			want:     []byte("header1,header2\nrow1,data1\nrow2,data2\n"),
		},
		"existing with newline": {
			existing: []byte("header1,header2\nrow1,data1\n"),
			new:      []byte("header1,header2\nrow2,data2\n"),
			want:     []byte("header1,header2\nrow1,data1\nrow2,data2\n"),
		},
		"empty existing": {
			existing: []byte(""),
			new:      []byte("header1,header2\nrow1,data1\n"),
			want:     []byte("row1,data1\n"),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			sw := &S3Writer{}
			got, err := sw.combineCSVData(tc.existing, tc.new)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if string(got) != string(tc.want) {
				t.Errorf("combineCSVData() = %q, want %q", string(got), string(tc.want))
			}
		})
	}
}
