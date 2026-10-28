package ytdlp

import (
	"bufio"
	"bytes"
	"encoding/json/jsontext"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// The fixtures in testdata/progress are real yt-dlp 2026.10.26 output, one
// "<pipe>\t<line>" per line plus a final "exit\t<code>", recorded against a
// local HTTP server once per progress implementation (each needs its own
// templates): <scenario>.{legacy,upstream,overall}.txt. Scenarios:
//   - two_formats: "-f 137+140" via --load-info-json, video then audio, Merger.
//   - hls, hls_concurrent (-N 4): hlsnative, 11 fragments, no declared size.
//   - fragment_retry: fragment 4 drops mid-body once and is retried.
//   - resume: a half-size .part exists ("Resuming download at byte").
//   - http_retry_resume: the connection drops at 40%, resumed with Range.
//   - http_retry_restart: same, but Range is ignored ("Unable to resume").
//   - unknown_total: no Content-Length.
//   - aria2c: --downloader aria2c (no per-chunk progress at all).
//   - already_downloaded: the merged file exists from a previous run.

type fixtureLine struct {
	pipe Pipe
	line []byte
}

func loadFixture(t *testing.T, name string) (lines []fixtureLine, exitOK bool) {
	t.Helper()
	f, err := os.Open(filepath.Join("testdata", "progress", name+".txt"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(nil, 1<<20)
	for sc.Scan() {
		pipe, line, _ := strings.Cut(sc.Text(), "\t")
		if pipe == "exit" {
			exitOK = line == "0"
			continue
		}
		lines = append(lines, fixtureLine{pipe: Pipe(pipe), line: []byte(line)})
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	return lines, exitOK
}

// sample is the implementation-neutral view of one progress callback.
type sample struct {
	percent    float64
	downloaded int64
	total      int64
	speed      int64
	phase      string // downloading, post_processing, finished
	full       any    // the full update, when comparable, for duplicate detection
}

func runLegacy(t *testing.T, name string) []sample {
	lines, _ := loadFixture(t, name+".legacy")
	var out []sample
	l := newLegacyProgress(func(p DownloadProgress) {
		phase := "downloading"
		switch p.Status {
		case StatusPostProcessing:
			phase = "post_processing"
		case StatusFinished:
			phase = "finished"
		}
		out = append(out, sample{p.Percent, p.DownloadedBytes, p.TotalBytes, p.Speed, phase, p})
	})
	for _, fl := range lines {
		if fl.pipe == PipeStdout {
			l.consume(fl.line)
		}
	}
	return out
}

func runUpstream(t *testing.T, name string) []sample {
	lines, _ := loadFixture(t, name+".upstream")
	var out []sample
	h := newProgressHandler(func(u ProgressUpdate) {
		phase := "downloading"
		if u.Status == ProgressStatusPostProcessing {
			phase = "post_processing"
		}
		// ProgressUpdate carries timestamps, so duplicates are not comparable.
		out = append(out, sample{u.Percent(), int64(u.DownloadedBytes), int64(u.TotalBytes), 0, phase, nil})
	})
	for _, fl := range lines {
		// Same as timestampWriter.flush for the stdout pipe.
		if v, ok := bytes.CutPrefix(fl.line, progressPrefix); ok && fl.pipe == PipeStdout {
			h.parse(jsontext.Value(v))
		}
	}
	return out
}

func runOverall(t *testing.T, name string) []OverallProgress {
	lines, exitOK := loadFixture(t, name+".overall")
	var out []OverallProgress
	tr := newOverallTracker(func(p OverallProgress) { out = append(out, p) })
	for _, fl := range lines {
		tr.consume(fl.line)
	}
	if exitOK {
		tr.finish()
	}
	return out
}

func overallSamples(updates []OverallProgress) []sample {
	out := make([]sample, 0, len(updates))
	for _, u := range updates {
		out = append(out, sample{u.Percent, u.Downloaded, u.Total, u.Speed, string(u.Phase), u})
	}
	return out
}

// Glitches a progress stream can have. Each one is something a UI shows.
const (
	glitchPercentBack    = "percent goes backwards"
	glitchDownloadedBack = "downloaded goes backwards"
	glitchShortTotal     = "final downloaded/total miss the requested bytes"
	glitchNoFinish       = "never reports finished/100%"
	glitchNoSpeed        = "speed never reported"
	glitchDuplicates     = "identical consecutive updates"
	glitchPhaseBack      = "phase goes backwards"
)

func glitches(samples []sample, wantBytes int64, wantSpeed bool) []string {
	var got []string
	add := func(g string) {
		if !slices.Contains(got, g) {
			got = append(got, g)
		}
	}
	rank := map[string]int{"downloading": 0, "post_processing": 1, "finished": 2}
	sawSpeed := false
	for i, s := range samples {
		sawSpeed = sawSpeed || s.speed > 0
		if i == 0 {
			continue
		}
		p := samples[i-1]
		if s.percent < p.percent {
			add(glitchPercentBack)
		}
		if s.downloaded < p.downloaded {
			add(glitchDownloadedBack)
		}
		if rank[s.phase] < rank[p.phase] {
			add(glitchPhaseBack)
		}
		if s.full != nil && s.full == p.full {
			add(glitchDuplicates)
		}
	}
	if len(samples) == 0 {
		return []string{glitchNoFinish}
	}
	last := samples[len(samples)-1]
	if last.phase != "finished" && last.percent < 100 {
		add(glitchNoFinish)
	}
	maxDownloaded := int64(0)
	for _, s := range samples {
		maxDownloaded = max(maxDownloaded, s.downloaded)
	}
	if wantBytes > 0 && maxDownloaded < wantBytes {
		add(glitchShortTotal)
	}
	if wantSpeed && !sawSpeed {
		add(glitchNoSpeed)
	}
	slices.Sort(got)
	return got
}

const (
	videoBytes = 4612157 // testdata video stream (format 137 in two_formats)
	audioBytes = 128355  // testdata audio stream (format 140)
	hlsBytes   = 8601000 // the 11 HLS fragments, remuxed
)

// TestProgressImplementations_RecordedGlitches characterizes the legacy
// SetProgressFn callback and upstream's ProgressFunc against real yt-dlp
// output, and requires OverallProgressFunc to have none of their glitches.
func TestProgressImplementations_RecordedGlitches(t *testing.T) {
	t.Parallel()

	tests := []struct {
		fixture   string
		wantBytes int64 // bytes every implementation should account for
		speed     bool  // whether yt-dlp reports a speed in this scenario
		legacy    []string
		upstream  []string
	}{
		{
			// bv+ba: legacy stops at the video and reports the audio and merge as
			// post-processing of the video; upstream restarts at 0% for the audio
			// and its merge updates carry 0%.
			fixture: "two_formats", wantBytes: videoBytes + audioBytes, speed: true,
			legacy:   []string{glitchDuplicates, glitchShortTotal, glitchNoSpeed},
			upstream: []string{glitchDownloadedBack, glitchNoFinish, glitchPercentBack, glitchShortTotal, glitchNoSpeed},
		},
		{
			// hlsnative only sends total_bytes_estimate, a float: the legacy decoder
			// cannot parse it into int64, so TotalBytes stays 0 and its percent
			// (yt-dlp's _percent_str) follows the swinging estimate and stops at
			// 98.8%. Upstream also follows the estimate. In every fixture
			// upstream's post-processing updates carry 0/0 bytes, and the legacy
			// speed is always 0 (strconv.Atoi on a float).
			fixture: "hls", wantBytes: hlsBytes, speed: true,
			legacy:   []string{glitchDuplicates, glitchNoSpeed, glitchPercentBack, glitchShortTotal},
			upstream: []string{glitchDownloadedBack, glitchNoFinish, glitchPercentBack, glitchNoSpeed},
		},
		{
			fixture: "hls_concurrent", wantBytes: hlsBytes, speed: true,
			legacy:   []string{glitchDuplicates, glitchNoSpeed, glitchPercentBack, glitchShortTotal},
			upstream: []string{glitchDownloadedBack, glitchNoFinish, glitchPercentBack, glitchNoSpeed},
		},
		{
			fixture: "fragment_retry", wantBytes: hlsBytes, speed: true,
			legacy:   []string{glitchDuplicates, glitchNoSpeed, glitchPercentBack, glitchShortTotal},
			upstream: []string{glitchDownloadedBack, glitchNoFinish, glitchPercentBack, glitchNoSpeed},
		},
		{
			fixture: "resume", wantBytes: videoBytes, speed: true,
			legacy:   []string{glitchDuplicates, glitchNoSpeed},
			upstream: []string{glitchDownloadedBack, glitchNoFinish, glitchPercentBack, glitchNoSpeed},
		},
		{
			fixture: "http_retry_resume", wantBytes: videoBytes, speed: true,
			legacy:   []string{glitchDuplicates, glitchNoSpeed},
			upstream: []string{glitchDownloadedBack, glitchNoFinish, glitchPercentBack, glitchNoSpeed},
		},
		{
			// The server ignores Range on retry: yt-dlp restarts from byte 0.
			fixture: "http_retry_restart", wantBytes: videoBytes, speed: true,
			legacy:   []string{glitchDownloadedBack, glitchDuplicates, glitchNoSpeed, glitchPercentBack},
			upstream: []string{glitchDownloadedBack, glitchNoFinish, glitchPercentBack, glitchNoSpeed},
		},
		{
			// No Content-Length: total stays 0 until the file finishes.
			fixture: "unknown_total", wantBytes: videoBytes, speed: true,
			legacy:   []string{glitchDuplicates, glitchNoSpeed},
			upstream: []string{glitchDownloadedBack, glitchNoFinish, glitchPercentBack, glitchNoSpeed},
		},
		{
			// External downloaders report a single "finished" per file.
			fixture: "aria2c", wantBytes: videoBytes,
			legacy:   []string{glitchDuplicates, glitchShortTotal},
			upstream: []string{glitchDownloadedBack, glitchNoFinish, glitchPercentBack},
		},
		{
			// The merged file exists: no download hooks at all.
			fixture:  "already_downloaded",
			legacy:   []string{},
			upstream: []string{glitchNoFinish},
		},
	}

	for _, tt := range tests {
		t.Run(tt.fixture, func(t *testing.T) {
			t.Parallel()

			check := func(impl string, samples []sample, want []string) {
				got := glitches(samples, tt.wantBytes, tt.speed)
				if want == nil {
					want = []string{}
				}
				slices.Sort(want)
				if !slices.Equal(got, want) && !(len(got) == 0 && len(want) == 0) {
					t.Errorf("%s glitches = %q, want %q", impl, got, want)
				}
			}
			check("legacy", runLegacy(t, tt.fixture), tt.legacy)
			check("upstream", runUpstream(t, tt.fixture), tt.upstream)

			overall := runOverall(t, tt.fixture)
			check("overall", overallSamples(overall), nil)
			assertOverallInvariants(t, overall)
		})
	}
}

// assertOverallInvariants checks the documented OverallProgress guarantees.
func assertOverallInvariants(t *testing.T, updates []OverallProgress) {
	t.Helper()
	if len(updates) == 0 {
		t.Fatal("no updates")
	}
	finished := 0
	for i, u := range updates {
		if u.Downloaded > u.Total {
			t.Errorf("update %d: downloaded %d > total %d", i, u.Downloaded, u.Total)
		}
		if u.Percent < 0 || u.Percent > 100 {
			t.Errorf("update %d: percent %v out of range", i, u.Percent)
		}
		if i > 0 && !u.TotalEstimated && !updates[i-1].TotalEstimated && u.Percent < 100 && u.Total != updates[i-1].Total {
			t.Errorf("update %d: exact total changed %d -> %d", i, updates[i-1].Total, u.Total)
		}
		if u.Phase == ProgressPhaseFinished {
			finished++
		}
	}
	if finished != 1 {
		t.Errorf("finished reported %d times, want 1", finished)
	}
	if last := updates[len(updates)-1]; last.Phase != ProgressPhaseFinished || last.Percent != 100 || last.Downloaded != last.Total {
		t.Errorf("last update = %+v, want finished at 100%% with downloaded == total", last)
	}
}

func TestOverallProgress_TwoFormats(t *testing.T) {
	t.Parallel()

	updates := runOverall(t, "two_formats")

	first := updates[0]
	if first.Phase != ProgressPhaseDownloading || first.FormatCount != 2 || first.Total != videoBytes+audioBytes+4000 || !first.TotalEstimated {
		t.Errorf("first update = %+v, want both declared sizes up front (audio is filesize_approx)", first)
	}

	var formats, processors []string
	for _, u := range updates {
		if u.FormatID != "" && !slices.Contains(formats, u.FormatID) {
			formats = append(formats, u.FormatID)
			if u.FormatIndex != len(formats) {
				t.Errorf("format %s index = %d, want %d", u.FormatID, u.FormatIndex, len(formats))
			}
		}
		if u.PostProcessor != "" && !slices.Contains(processors, u.PostProcessor) {
			processors = append(processors, u.PostProcessor)
		}
	}
	if !slices.Equal(formats, []string{"137", "140"}) {
		t.Errorf("formats = %v, want [137 140]", formats)
	}
	if !slices.Equal(processors, []string{"Merger", "Metadata", "MoveFiles"}) {
		t.Errorf("post-processors = %v", processors)
	}

	// The audio download continues from the video's bytes instead of restarting.
	for _, u := range updates {
		if u.FormatID == "140" && u.Downloaded < videoBytes {
			t.Errorf("audio update restarted overall bytes: %+v", u)
		}
	}
	last := updates[len(updates)-1]
	if last.Total != videoBytes+audioBytes || last.TotalEstimated {
		t.Errorf("final total = %d (estimated %v), want exact %d", last.Total, last.TotalEstimated, videoBytes+audioBytes)
	}
}

func TestOverallProgress_HLSEstimateIsSteady(t *testing.T) {
	t.Parallel()

	// yt-dlp's own total_bytes_estimate in this recording swings from 9.4MB to
	// 19MB (actual: 8.6MB). The tracker's percent must climb steadily, never
	// sitting far below the byte progress.
	updates := runOverall(t, "hls")
	var downloading []OverallProgress
	for _, u := range updates {
		if u.Phase == ProgressPhaseDownloading && u.Percent < 100 {
			downloading = append(downloading, u)
		}
	}
	if len(downloading) < 10 {
		t.Fatalf("only %d downloading updates", len(downloading))
	}
	last := downloading[len(downloading)-1]
	if real := float64(last.Downloaded) / hlsBytes * 100; last.Percent < real-15 {
		t.Errorf("last downloading percent %.1f, real progress %.1f", last.Percent, real)
	}
	for _, u := range downloading {
		if u.Total > 2*hlsBytes {
			t.Errorf("estimate %d is more than twice the real size", u.Total)
		}
	}
}
