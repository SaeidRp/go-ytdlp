package ytdlp

import (
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// TestCommand_SetProgressFn keeps the deprecated callback that storage uses
// working on top of upstream's command/writer.
func TestCommand_SetProgressFn(t *testing.T) {
	t.Parallel()

	server := newMockServer(t, "testdata/sample-1.mp4")

	var mu sync.Mutex
	var statuses []DownloadStatus
	result, err := New().
		NoUpdate().
		Output(filepath.Join(t.TempDir(), "%(id)s.%(ext)s")).
		SetProgressFn(func(p DownloadProgress) {
			mu.Lock()
			statuses = append(statuses, p.Status)
			mu.Unlock()
		}).
		Run(t.Context(), server.fileURL)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(result.Stdout, progressLinePrefix+"ID=") {
		t.Error("progress lines leaked into Result.Stdout")
	}
	if len(result.Downloads) != 1 || result.Downloads[0].Status != StatusFinished || result.Downloads[0].TotalBytes != 4745623 {
		t.Errorf("downloads = %+v", result.Downloads)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(statuses) < 3 || statuses[0] != StatusStarting || statuses[len(statuses)-1] != StatusFinished {
		t.Errorf("statuses = %v", statuses)
	}
}
