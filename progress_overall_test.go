package ytdlp

import (
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	evBefore2 = `go-ytdlp-progress:{"event":"before_dl","info":{"id":"v","format_id":"137+140"},"formats":[{"format_id":"137","filesize":1000},{"format_id":"140","filesize":200}]}`
	evAfter   = `go-ytdlp-progress:{"event":"after_video","info":{"id":"v"}}`
)

func evDL(format, status string, downloaded, total int) string {
	return `go-ytdlp-progress:{"event":"download","progress":{"status":"` + status + `","downloaded_bytes":` + strconv.Itoa(downloaded) +
		`,"total_bytes":` + strconv.Itoa(total) + `,"speed":100},"info":{"id":"v","format_id":"` + format + `"}}`
}

func evEst(format string, downloaded int, estimate string, fragIndex, fragCount int) string {
	return `go-ytdlp-progress:{"event":"download","progress":{"status":"downloading","downloaded_bytes":` + strconv.Itoa(downloaded) +
		`,"total_bytes_estimate":` + estimate + `,"fragment_index":` + strconv.Itoa(fragIndex) + `,"fragment_count":` + strconv.Itoa(fragCount) +
		`},"info":{"id":"v","format_id":"` + format + `"}}`
}

func evPP(name, status string) string {
	return `go-ytdlp-progress:{"event":"postprocess","progress":{"status":"` + status + `","postprocessor":"` + name + `"},"info":{"id":"v"}}`
}

func feedOverall(lines ...string) []OverallProgress {
	var out []OverallProgress
	tr := newOverallTracker(func(p OverallProgress) { out = append(out, p) })
	for _, l := range lines {
		tr.consume([]byte(l))
	}
	return out
}

type want struct {
	phase      ProgressPhase
	downloaded int64
	total      int64
	percent    float64
	format     string
}

func TestOverallTracker(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		lines []string
		want  []want // expected updates, in order
	}{
		{
			name: "two formats in sequence add up",
			lines: []string{
				evBefore2,
				evDL("137", "downloading", 500, 1000),
				evDL("137", "finished", 1000, 1000),
				evDL("140", "downloading", 100, 200),
				evDL("140", "finished", 200, 200),
				evPP("Merger", "started"), evPP("Merger", "finished"),
				evAfter,
			},
			want: []want{
				{ProgressPhaseDownloading, 0, 1200, 0, ""},
				{ProgressPhaseDownloading, 500, 1200, 500.0 / 12, "137"},
				{ProgressPhaseDownloading, 1000, 1200, 1000.0 / 12, "137"},
				{ProgressPhaseDownloading, 1100, 1200, 1100.0 / 12, "140"},
				{ProgressPhaseDownloading, 1200, 1200, 100, "140"},
				{ProgressPhasePostProcessing, 1200, 1200, 100, ""},
				{ProgressPhaseFinished, 1200, 1200, 100, ""},
			},
		},
		{
			name: "restart from byte 0 does not go backwards",
			lines: []string{
				evBefore2,
				evDL("137", "downloading", 600, 1000),
				evDL("137", "downloading", 10, 1000), // "Unable to resume"
				evDL("137", "downloading", 700, 1000),
			},
			want: []want{
				{ProgressPhaseDownloading, 0, 1200, 0, ""},
				{ProgressPhaseDownloading, 600, 1200, 50, "137"},
				{ProgressPhaseDownloading, 700, 1200, 700.0 / 12, "137"},
			},
		},
		{
			name: "garbage, truncated, NaN and interleaved lines",
			lines: []string{
				evBefore2,
				`go-ytdlp-progress:{"event":"download","progress":{"status":"downl`,
				`go-ytdlp-progress:not json`,
				`go-ytdlp-progress:{"event":"download","progress":{"downloaded_bytes":NaN},"info":{"id":"v","format_id":"137"}}`,
				"\r[#2089b0 400KiB/33MiB(1%) CN:1 DL:115KiB]\r" + evDL("137", "downloading", 250, 1000) + ` trailing`,
				`[download] go-ytdlp-progress: in a title is not an event`,
			},
			want: []want{
				{ProgressPhaseDownloading, 0, 1200, 0, ""},
				{ProgressPhaseDownloading, 250, 1200, 250.0 / 12, "137"},
			},
		},
		{
			name: "identical lines are emitted once",
			lines: []string{
				evBefore2,
				evDL("137", "downloading", 500, 1000),
				evDL("137", "downloading", 500, 1000),
				evPP("Merger", "started"), // ignored: formats not finished
				evAfter, evAfter,
			},
			want: []want{
				{ProgressPhaseDownloading, 0, 1200, 0, ""},
				{ProgressPhaseDownloading, 500, 1200, 500.0 / 12, "137"},
				{ProgressPhaseFinished, 1200, 1200, 100, ""},
			},
		},
		{
			name: "post-processors before the download and subtitle downloads are ignored",
			lines: []string{
				evPP("SponsorBlock", "started"),
				`go-ytdlp-progress:{"event":"download","progress":{"status":"finished","downloaded_bytes":50,"total_bytes":50},"info":{}}`,
				evBefore2,
				evPP("ThumbnailsConvertor", "started"),
				evDL("137", "downloading", 100, 1000),
			},
			want: []want{
				{ProgressPhaseDownloading, 0, 1200, 0, ""},
				{ProgressPhaseDownloading, 100, 1200, 100.0 / 12, "137"},
			},
		},
		{
			// --quiet routes post-processor lines to stderr, so they can be read
			// before the last stdout download line.
			name: "reordered lines across pipes",
			lines: []string{
				evBefore2,
				evDL("137", "finished", 1000, 1000),
				evPP("Merger", "started"), // before 140 finished: ignored
				evDL("140", "finished", 200, 200),
				evPP("MoveFiles", "started"),
				evDL("140", "downloading", 150, 200), // late: ignored
				evAfter,
			},
			want: []want{
				{ProgressPhaseDownloading, 0, 1200, 0, ""},
				{ProgressPhaseDownloading, 1000, 1200, 1000.0 / 12, "137"},
				{ProgressPhaseDownloading, 1200, 1200, 100, "140"},
				{ProgressPhasePostProcessing, 1200, 1200, 100, ""},
				{ProgressPhaseFinished, 1200, 1200, 100, ""},
			},
		},
		{
			name: "estimated format that turns out smaller",
			lines: []string{
				`go-ytdlp-progress:{"event":"before_dl","info":{"id":"v","format_id":"1+2"},"formats":[{"format_id":"1","filesize":1000},{"format_id":"2","filesize_approx":1000}]}`,
				evDL("1", "finished", 1000, 1000),
				evDL("2", "downloading", 100, 500), // Content-Length: half the estimate
				evDL("2", "finished", 500, 500),
			},
			want: []want{
				{ProgressPhaseDownloading, 0, 2000, 0, ""},
				{ProgressPhaseDownloading, 1000, 2000, 50, "1"},
				{ProgressPhaseDownloading, 1100, 1500, 1100.0 / 15, "2"},
				{ProgressPhaseDownloading, 1500, 1500, 100, "2"},
			},
		},
		{
			name: "unknown size of a format weighs formats equally",
			lines: []string{
				`go-ytdlp-progress:{"event":"before_dl","info":{"id":"v","format_id":"1+2"},"formats":[{"format_id":"1","filesize":1000},{"format_id":"2"}]}`,
				evDL("1", "finished", 1000, 1000),
				`go-ytdlp-progress:{"event":"download","progress":{"status":"downloading","downloaded_bytes":300},"info":{"id":"v","format_id":"2"}}`,
				evDL("2", "finished", 600, 600),
			},
			want: []want{
				{ProgressPhaseDownloading, 0, 1000, 0, ""},
				{ProgressPhaseDownloading, 1000, 1000, 50, "1"},
				{ProgressPhaseDownloading, 1300, 1300, 50, "2"},
				{ProgressPhaseDownloading, 1600, 1600, 100, "2"},
			},
		},
		{
			// yt-dlp's estimate swings 2x; ours only changes per completed fragment.
			name: "fragment estimate",
			lines: []string{
				`go-ytdlp-progress:{"event":"before_dl","info":{"id":"v","format_id":"hls"},"formats":[]}`,
				evEst("hls", 100, "1000.0", 0, 10),
				evEst("hls", 300, "3000.5", 0, 10),
				evEst("hls", 300, "1500.0", 3, 10), // 3 fragments done: 300/3*10
				evEst("hls", 350, "9000.0", 3, 10),
			},
			want: []want{
				{ProgressPhaseDownloading, 0, 0, 0, ""},
				{ProgressPhaseDownloading, 100, 1000, 10, "hls"},
				{ProgressPhaseDownloading, 300, 3000, 10, "hls"},
				{ProgressPhaseDownloading, 300, 1000, 30, "hls"},
				{ProgressPhaseDownloading, 350, 1000, 35, "hls"},
			},
		},
		{
			name: "merging downloader reports the combined format",
			lines: []string{
				evBefore2,
				evDL("137+140", "finished", 1150, 1150),
				evAfter,
			},
			want: []want{
				{ProgressPhaseDownloading, 0, 1200, 0, ""},
				{ProgressPhaseDownloading, 1150, 1150, 100, "137+140"},
				{ProgressPhaseFinished, 1150, 1150, 100, ""},
			},
		},
		{
			name: "already downloaded file finishes without download events",
			lines: []string{
				evBefore2,
				evPP("MoveFiles", "started"),
				evAfter,
			},
			want: []want{
				{ProgressPhaseDownloading, 0, 1200, 0, ""},
				{ProgressPhaseFinished, 1200, 1200, 100, ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := feedOverall(tt.lines...)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d updates, want %d:\n%+v", len(got), len(tt.want), got)
			}
			for i, w := range tt.want {
				g := got[i]
				if g.Phase != w.phase || g.Downloaded != w.downloaded || g.Total != w.total ||
					!approx(g.Percent, w.percent) || g.FormatID != w.format {
					t.Errorf("update %d = %+v, want %+v", i, g, w)
				}
			}
		})
	}
}

func approx(a, b float64) bool { return a-b < 1e-9 && b-a < 1e-9 }

func TestOverallTracker_ConsumesOnlyPrefixedLines(t *testing.T) {
	t.Parallel()

	tr := newOverallTracker(func(OverallProgress) {})
	for line, want := range map[string]bool{
		"[download] Destination: x.mp4":   false,
		"dl:ID=\"x\"|Status=\"starting\"": false,
		"progress:{}":                     false,
		"go-ytdlp-progress:{broken":       true, // ours, malformed: still kept out of the logs
		evAfter:                           true,
	} {
		if got := tr.consume([]byte(line)); got != want {
			t.Errorf("consume(%q) = %v, want %v", line, got, want)
		}
	}
}

func TestOverallProgressFunc_Flags(t *testing.T) {
	t.Parallel()

	fn := func(OverallProgress) {}
	cmd := New().
		SetProgressFn(func(DownloadProgress) {}).
		ProgressFunc(time.Second, func(ProgressUpdate) {}).
		Print("after_move:%(filepath)s").
		OverallProgressFunc(0, fn)

	vs := cmd.GetFlagConfig().VerbositySimulation
	if !slices.Equal(vs.ProgressTemplate, []string{overallDownloadTemplate, overallPostProcessTemplate}) {
		t.Errorf("progress templates = %q", vs.ProgressTemplate)
	}
	if !slices.Equal(vs.Print, []string{"after_move:%(filepath)s", overallBeforeDownloadPrint, overallAfterVideoPrint}) {
		t.Errorf("print templates = %q", vs.Print)
	}
	if vs.NoQuiet == nil || vs.Newline == nil || vs.Progress == nil || vs.ProgressDelta == nil || *vs.ProgressDelta != 0.1 {
		t.Errorf("flags = %+v", vs)
	}
	if cmd.progress != nil || cmd.legacyProgress != nil || cmd.overallProgress == nil {
		t.Error("OverallProgressFunc must replace the other progress handlers")
	}

	if q := New().Quiet().OverallProgressFunc(time.Second, fn).GetFlagConfig().VerbositySimulation; q.NoQuiet != nil || q.Quiet == nil {
		t.Error("an explicit Quiet must be kept")
	}

	cmd.UnsetOverallProgressFunc()
	vs = cmd.GetFlagConfig().VerbositySimulation
	if len(vs.ProgressTemplate) != 0 || !slices.Equal(vs.Print, []string{"after_move:%(filepath)s"}) || cmd.overallProgress != nil {
		t.Errorf("after unset: templates %q, print %q", vs.ProgressTemplate, vs.Print)
	}
}

func TestCommand_OverallProgress(t *testing.T) {
	t.Parallel()

	server := newMockServer(t, "testdata/sample-1.mp4")
	dir := t.TempDir()

	var mu sync.Mutex
	var updates []OverallProgress

	result, err := New().
		NoUpdate().
		ForceOverwrites().
		RecodeVideo("mkv").
		Output(filepath.Join(dir, "%(extractor)s - %(title)s.%(ext)s")).
		OverallProgressFunc(100*time.Millisecond, func(p OverallProgress) {
			mu.Lock()
			updates = append(updates, p)
			mu.Unlock()
		}).
		Run(t.Context(), server.fileURL)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(result.Stdout, overallPrefix) {
		t.Error("progress lines leaked into Result.Stdout")
	}

	mu.Lock()
	defer mu.Unlock()
	assertOverallInvariants(t, updates)

	var phases []ProgressPhase
	for _, u := range updates {
		if !slices.Contains(phases, u.Phase) {
			phases = append(phases, u.Phase)
		}
	}
	if !slices.Equal(phases, []ProgressPhase{ProgressPhaseDownloading, ProgressPhasePostProcessing, ProgressPhaseFinished}) {
		t.Errorf("phases = %v", phases)
	}
	if last := updates[len(updates)-1]; last.Total != 4745623 { // testdata/sample-1.mp4
		t.Errorf("final total = %d, want 4745623", last.Total)
	}
}
