package output

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// faultyFile wraps an *os.File so tests can inject Write and Close failures
// without depending on OS-specific filesystem behaviour. The underlying file is
// always really closed so the temp directory can be cleaned up.
type faultyFile struct {
	*os.File
	writeErr error // returned by Write when non-nil (nothing is written)
	closeErr error // returned by Close when non-nil, after closing the real file
}

func (f *faultyFile) Write(p []byte) (int, error) {
	if f.writeErr != nil {
		return 0, f.writeErr
	}
	return f.File.Write(p)
}

func (f *faultyFile) Close() error {
	err := f.File.Close()
	if f.closeErr != nil {
		return f.closeErr
	}
	return err
}

// TestFileWriterReportsCloseErrors covers T-1399: every FileWriter write path —
// overwrite, byte-level append, CSV append, and the HTML temp-file append — must
// report a Close failure when no earlier error occurred (on some filesystems
// Close is where delayed writeback or metadata errors surface), and must never
// let a Close failure mask an earlier write error.
func TestFileWriterReportsCloseErrors(t *testing.T) {
	errClose := errors.New("delayed writeback failed at close")
	errWrite := errors.New("write failed")

	const htmlPage = "<html><body><p>first</p>" + HTMLAppendMarker + "</body></html>"

	tests := map[string]struct {
		appendMode  bool
		format      string
		existing    string // initial file content; empty means the file does not exist yet
		data        []byte
		writeErr    error
		closeErr    error
		wantErr     error  // must be in the returned error chain; nil means Write must succeed
		wantNotErr  error  // must not be in the returned error chain
		wantMsg     string // substring the error message must contain
		wantContent string // file content after Write; empty means not checked
	}{
		"wrapped file without faults appends normally": {
			appendMode:  true,
			format:      FormatJSON,
			existing:    `{"a":1}` + "\n",
			data:        []byte(`{"b":2}` + "\n"),
			wantContent: `{"a":1}` + "\n" + `{"b":2}` + "\n",
		},
		"overwrite path reports close error": {
			format:   FormatJSON,
			data:     []byte(`{"a":1}`),
			closeErr: errClose,
			wantErr:  errClose,
			wantMsg:  "failed to close file",
		},
		"byte-level append reports close error": {
			appendMode: true,
			format:     FormatJSON,
			existing:   `{"a":1}` + "\n",
			data:       []byte(`{"b":2}` + "\n"),
			closeErr:   errClose,
			wantErr:    errClose,
			wantMsg:    "failed to close file",
		},
		"csv append reports close error": {
			appendMode: true,
			format:     FormatCSV,
			existing:   "a,b\n1,2\n",
			data:       []byte("a,b\n3,4\n"),
			closeErr:   errClose,
			wantErr:    errClose,
			wantMsg:    "failed to close file",
		},
		"html append reports temp file close error and leaves original intact": {
			appendMode:  true,
			format:      FormatHTML,
			existing:    htmlPage,
			data:        []byte("<p>second</p>"),
			closeErr:    errClose,
			wantErr:     errClose,
			wantMsg:     "failed to close temp file",
			wantContent: htmlPage,
		},
		"overwrite path keeps write error when close also fails": {
			format:     FormatJSON,
			data:       []byte(`{"a":1}`),
			writeErr:   errWrite,
			closeErr:   errClose,
			wantErr:    errWrite,
			wantNotErr: errClose,
		},
		"byte-level append keeps write error when close also fails": {
			appendMode: true,
			format:     FormatJSON,
			existing:   `{"a":1}` + "\n",
			data:       []byte(`{"b":2}` + "\n"),
			writeErr:   errWrite,
			closeErr:   errClose,
			wantErr:    errWrite,
			wantNotErr: errClose,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			filename := "out." + tc.format
			if tc.existing != "" {
				if err := os.WriteFile(filepath.Join(dir, filename), []byte(tc.existing), 0644); err != nil {
					t.Fatalf("failed to create existing file: %v", err)
				}
			}

			var opts []FileWriterOption
			if tc.appendMode {
				opts = append(opts, WithAppendMode())
			}
			fw, err := NewFileWriterWithOptions(dir, filename, opts...)
			if err != nil {
				t.Fatalf("NewFileWriterWithOptions() error = %v", err)
			}
			fw.wrapFile = func(f *os.File) writableFile {
				return &faultyFile{File: f, writeErr: tc.writeErr, closeErr: tc.closeErr}
			}

			err = fw.Write(context.Background(), tc.format, tc.data)

			if tc.wantErr == nil {
				if err != nil {
					t.Fatalf("Write() error = %v, want nil", err)
				}
			} else {
				if err == nil {
					t.Fatalf("Write() error = nil, want %v", tc.wantErr)
				}
				if !errors.Is(err, tc.wantErr) {
					t.Errorf("Write() error = %v, want chain to contain %v", err, tc.wantErr)
				}
				if tc.wantNotErr != nil && errors.Is(err, tc.wantNotErr) {
					t.Errorf("Write() error = %v, want chain NOT to contain %v", err, tc.wantNotErr)
				}
				if tc.wantMsg != "" && !strings.Contains(err.Error(), tc.wantMsg) {
					t.Errorf("Write() error = %q, want it to contain %q", err.Error(), tc.wantMsg)
				}
			}

			if tc.wantContent != "" {
				got, readErr := os.ReadFile(filepath.Join(dir, filename))
				if readErr != nil {
					t.Fatalf("failed to read output file: %v", readErr)
				}
				if string(got) != tc.wantContent {
					t.Errorf("file content = %q, want %q", string(got), tc.wantContent)
				}
			}

			// The HTML path writes through a temp file; a failed append must not leak it.
			leftovers, _ := filepath.Glob(filepath.Join(dir, ".go-output-*.tmp"))
			if len(leftovers) != 0 {
				t.Errorf("temp files left behind = %v, want none", leftovers)
			}
		})
	}
}
