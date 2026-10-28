package ytdlp

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"math"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

// overallPrefix marks the machine-readable lines emitted by the templates
// below. Every value is rendered with yt-dlp's "j" conversion (json.dumps with
// ensure_ascii), so ids, titles and file names can never break the line: quotes,
// backslashes and newlines are escaped and non-ASCII becomes \uXXXX.
const overallPrefix = "go-ytdlp-progress:"

var (
	// Download hooks (yt-dlp downloader/common.py report_progress): printed for
	// "downloading" (throttled by --progress-delta) and always for "finished".
	overallDownloadTemplate = "download:" + overallPrefix + `{"event":"download",` +
		`"progress":%(progress.{status,downloaded_bytes,total_bytes,total_bytes_estimate,eta,speed,fragment_index,fragment_count}|null)j,` +
		`"info":%(info.{id,format_id,playlist_index}|null)j}`

	// Post-processor hooks (postprocessor/common.py): "started"/"finished" per
	// post-processor (Merger, EmbedThumbnail, MoveFiles, ...).
	overallPostProcessTemplate = "postprocess:" + overallPrefix + `{"event":"postprocess",` +
		`"progress":%(progress.{status,postprocessor}|null)j,` +
		`"info":%(info.{id,playlist_index}|null)j}`

	// before_dl runs once per video before any format is downloaded and is the
	// only point where requested_formats and their declared sizes are visible:
	// each sequential format download gets an info dict without them.
	overallBeforeDownloadPrint = "before_dl:" + overallPrefix + `{"event":"before_dl",` +
		`"info":%(.{id,format_id,filesize,filesize_approx,playlist_index,n_entries}|null)j,` +
		`"formats":%(requested_formats.:.{format_id,filesize,filesize_approx}|null)j}`

	// after_video runs once the video is downloaded, post-processed and moved.
	overallAfterVideoPrint = "after_video:" + overallPrefix + `{"event":"after_video",` +
		`"info":%(.{id,playlist_index}|null)j}`
)

// ProgressPhase is the phase of an [OverallProgress] update. Phases only move
// forward: downloading, then post-processing, then finished.
type ProgressPhase string

const (
	// ProgressPhaseDownloading covers every requested format of the video.
	ProgressPhaseDownloading ProgressPhase = "downloading"
	// ProgressPhasePostProcessing is entered once every format finished and a
	// post-processor (Merger, EmbedThumbnail, FixupM4a, MoveFiles, ...) starts.
	// yt-dlp reports no byte progress for post-processors.
	ProgressPhasePostProcessing ProgressPhase = "post_processing"
	// ProgressPhaseFinished is reported once per video, after post-processing.
	ProgressPhaseFinished ProgressPhase = "finished"
)

// OverallProgress is a snapshot of one video's progress across all of its
// requested formats (e.g. both streams of "bv+ba").
//
// Guarantees, per video:
//   - Phase never moves backwards; Finished is reported once.
//   - Percent never decreases and is 100 once every format has finished.
//     It is the value to display.
//   - Downloaded never decreases (a download restarted from byte 0 does not
//     lower it) and Downloaded <= Total.
//   - While TotalEstimated is false, Total is exact and constant. While it is
//     true, Total follows the current estimate and may move either way; for
//     fragmented (HLS/DASH) formats it is re-estimated once per completed
//     fragment from the average fragment size, which is far steadier than
//     yt-dlp's own total_bytes_estimate. When every format has finished, Total
//     and Downloaded become the exact sum of the downloaded files.
//   - Two consecutive callbacks are never identical.
type OverallProgress struct {
	// ID is the video id (yt-dlp's "id").
	ID string `json:"id"`
	// PlaylistIndex is the 1-based index in a playlist, 0 if not a playlist.
	PlaylistIndex int `json:"playlist_index,omitempty"`
	// PlaylistCount is the number of playlist entries being downloaded.
	PlaylistCount int `json:"playlist_count,omitempty"`

	Phase ProgressPhase `json:"phase"`

	// FormatID, FormatIndex (1-based) and FormatCount identify the format that
	// produced this update while downloading. FormatID is empty before the
	// first format starts.
	FormatID    string `json:"format_id,omitempty"`
	FormatIndex int    `json:"format_index,omitempty"`
	FormatCount int    `json:"format_count"`

	// PostProcessor is the running post-processor (e.g. "Merger") while Phase
	// is [ProgressPhasePostProcessing].
	PostProcessor string `json:"postprocessor,omitempty"`

	// Downloaded is the number of bytes downloaded across all formats.
	Downloaded int64 `json:"downloaded"`
	// Total is the expected size across all formats, 0 if nothing is known.
	Total int64 `json:"total"`
	// TotalEstimated is true unless every format's size is exact (known from
	// filesize, Content-Length, or a finished download).
	TotalEstimated bool `json:"total_estimated"`
	// Percent is 0-100.
	Percent float64 `json:"percent"`
	// Speed is the current speed in bytes per second, 0 if unknown.
	Speed int64 `json:"speed"`
	// ETA is the estimated time left for the remaining bytes of all formats.
	ETA time.Duration `json:"eta"`
}

// OverallProgressCallbackFunc receives [OverallProgress] updates. Calls are
// serialized and happen on yt-dlp's output pipe; the function must not block.
type OverallProgressCallbackFunc func(OverallProgress)

// OverallProgressFunc registers fn for one monotonic progress per video
// across all requested formats and post-processing. It is the replacement
// for [Command.SetProgressFn], and is an alternative to [Command.ProgressFunc]
// (per-file updates): setting one of them clears the others.
//
// frequency is passed to --progress-delta (minimum 100ms). The function also
// sets --progress, --newline, two --progress-template and two --print
// templates, and --no-quiet unless [Command.Quiet] was set (so all progress
// lines share stdout and stay ordered). Throttling the callback further is up
// to the caller.
func (c *Command) OverallProgressFunc(frequency time.Duration, fn OverallProgressCallbackFunc) *Command {
	c.clearProgressHandlers()
	if fn == nil {
		return c
	}
	if frequency < 100*time.Millisecond {
		frequency = 100 * time.Millisecond
	}

	c.Progress().
		ProgressDelta(frequency.Seconds()).
		ProgressTemplate(overallDownloadTemplate).
		ProgressTemplate(overallPostProcessTemplate).
		Print(overallBeforeDownloadPrint).
		Print(overallAfterVideoPrint).
		Newline()
	if c.flagConfig.VerbositySimulation.Quiet == nil {
		// --print implies --quiet, which would move post-processor lines to stderr.
		c.NoQuiet()
	}

	c.mu.Lock()
	c.overallProgress = fn
	c.mu.Unlock()
	return c
}

// UnsetOverallProgressFunc unsets the function set by [Command.OverallProgressFunc].
func (c *Command) UnsetOverallProgressFunc() *Command {
	c.clearProgressHandlers()
	return c
}

// clearProgressHandlers removes every progress callback and the templates the
// progress setters add, as they share yt-dlp's single "download:" template.
func (c *Command) clearProgressHandlers() {
	legacy := []string{}
	for _, f := range []func() (string, error){
		GetDownloadProgressTemplate, GetProgressBeforeDownloadTemplate,
		GetProgressPostProcessTemplate, GetProgressVideoDownloadedTemplate,
	} {
		if t, err := f(); err == nil {
			legacy = append(legacy, t)
		}
	}
	owned := func(t string) bool {
		return strings.Contains(t, overallPrefix) || strings.Contains(t, string(progressPrefix)+progressFormat) ||
			slices.Contains(legacy, t)
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	vs := &c.flagConfig.VerbositySimulation
	vs.ProgressTemplate = slices.DeleteFunc(vs.ProgressTemplate, owned)
	vs.Print = slices.DeleteFunc(vs.Print, owned)
	c.progress = nil
	c.legacyProgress = nil
	c.overallProgress = nil
}

type overallEvent struct {
	Event    string `json:"event"`
	Progress struct {
		Status             string   `json:"status"`
		DownloadedBytes    *float64 `json:"downloaded_bytes"`
		TotalBytes         *float64 `json:"total_bytes"`
		TotalBytesEstimate *float64 `json:"total_bytes_estimate"`
		ETA                *float64 `json:"eta"`
		Speed              *float64 `json:"speed"`
		FragmentIndex      *float64 `json:"fragment_index"`
		FragmentCount      *float64 `json:"fragment_count"`
		PostProcessor      string   `json:"postprocessor"`
	} `json:"progress"`
	Info    overallEventFormat   `json:"info"`
	Formats []overallEventFormat `json:"formats"`
}

type overallEventFormat struct {
	ID             string   `json:"id"`
	FormatID       string   `json:"format_id"`
	Filesize       *float64 `json:"filesize"`
	FilesizeApprox *float64 `json:"filesize_approx"`
	PlaylistIndex  *float64 `json:"playlist_index"`
	NEntries       *float64 `json:"n_entries"`
}

func (f *overallEventFormat) key() string {
	return f.ID + "\x00" + strconv.Itoa(toInt(f.PlaylistIndex))
}

type overallFormat struct {
	id            string
	declared      int64 // filesize / filesize_approx from the extractor
	declaredExact bool
	reported      int64 // total_bytes / total_bytes_estimate from the downloader
	reportedExact bool
	fragEstimate  int64 // average completed fragment size * fragment count
	fragIndex     int64
	downloaded    int64 // high-water mark; a restarted download does not lower it
	finished      bool
}

// size returns the best known size (never below the downloaded bytes) and
// whether it is exact. known is false when nothing but downloaded bytes exist.
func (f *overallFormat) size() (size int64, exact, known bool) {
	switch {
	case f.finished:
		return f.downloaded, true, true
	case f.reportedExact:
		return max(f.reported, f.downloaded), true, true
	case f.declaredExact:
		return max(f.declared, f.downloaded), true, true
	}
	// yt-dlp's fragment estimate (fragment.py) double-counts the in-flight
	// fragment and swings by 2x; prefer our per-fragment estimate, then the
	// extractor's filesize_approx.
	switch {
	case f.fragEstimate > 0:
		return max(f.fragEstimate, f.downloaded), false, true
	case f.declared > 0:
		return max(f.declared, f.downloaded), false, true
	}
	return max(f.reported, f.downloaded), false, f.reported > 0
}

type overallItem struct {
	formats    []*overallFormat
	combinedID string // e.g. "137+140", reported by downloaders that merge (ffmpeg)
	last       OverallProgress
	emitted    bool
}

// overallTracker folds the template lines into [OverallProgress] updates.
// It is fed from both stdout and stderr, hence the mutex.
type overallTracker struct {
	mu    sync.Mutex
	fn    OverallProgressCallbackFunc
	items map[string]*overallItem // ponytail: never evicted, one small entry per video per run
}

func newOverallTracker(fn OverallProgressCallbackFunc) *overallTracker {
	return &overallTracker{fn: fn, items: make(map[string]*overallItem)}
}

// consume handles one output line. It reports whether the line carried the
// progress prefix (and must therefore not be logged). The prefix may appear
// mid-line: external downloaders (aria2c) share stdout and write \r-only
// status bars, which end up in front of the next line.
func (t *overallTracker) consume(line []byte) bool {
	i := bytes.Index(line, []byte(overallPrefix))
	if i < 0 {
		return false
	}

	var ev overallEvent
	raw, err := jsontext.NewDecoder(bytes.NewReader(line[i+len(overallPrefix):])).ReadValue()
	if err != nil || json.Unmarshal(raw, &ev) != nil {
		return true // truncated or malformed: drop it.
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	t.handle(&ev)
	return true
}

func (t *overallTracker) handle(ev *overallEvent) {
	key := ev.Info.key()

	switch ev.Event {
	case "before_dl":
		if ev.Info.ID == "" {
			return
		}
		// Every before_dl starts a fresh item (e.g. "-f 18,22" downloads the same
		// video twice, each with its own before_dl).
		item := &overallItem{combinedID: ev.Info.FormatID}
		formats := ev.Formats
		if len(formats) == 0 {
			formats = []overallEventFormat{ev.Info}
		}
		for _, f := range formats {
			item.formats = append(item.formats, newOverallFormat(f))
		}
		t.items[key] = item
		t.emit(item, ev, ProgressPhaseDownloading, nil)

	case "download":
		item := t.items[key]
		if item == nil {
			// Subtitles are downloaded before before_dl with an info dict that has
			// no id/format_id; skip those. Otherwise start tracking lazily.
			if ev.Info.ID == "" || ev.Info.FormatID == "" {
				return
			}
			item = &overallItem{}
			t.items[key] = item
		}
		if item.last.Phase != "" && item.last.Phase != ProgressPhaseDownloading {
			return // late or reordered line.
		}

		f := item.format(ev.Info.FormatID)
		p := &ev.Progress
		if p.TotalBytes != nil {
			f.reported, f.reportedExact = toInt64(p.TotalBytes), true
		} else if p.TotalBytesEstimate != nil && !f.reportedExact {
			f.reported = toInt64(p.TotalBytesEstimate)
		}
		if p.DownloadedBytes != nil {
			f.downloaded = max(f.downloaded, toInt64(p.DownloadedBytes))
		}
		// fragment_index counts completed fragments; at the moment it advances,
		// downloaded_bytes is (mostly) whole fragments.
		if idx, n := toInt64(p.FragmentIndex), toInt64(p.FragmentCount); idx > f.fragIndex && n > 0 && p.DownloadedBytes != nil {
			f.fragIndex = idx
			f.fragEstimate = int64(float64(toInt64(p.DownloadedBytes)) / float64(idx) * float64(n))
		}
		if p.Status == "finished" {
			// "already downloaded" files report total_bytes only.
			if p.DownloadedBytes != nil {
				f.downloaded = toInt64(p.DownloadedBytes)
			} else if p.TotalBytes != nil {
				f.downloaded = toInt64(p.TotalBytes)
			}
			f.finished = true
		}
		t.emit(item, ev, ProgressPhaseDownloading, f)

	case "postprocess":
		item := t.items[key]
		if item == nil || item.last.Phase == ProgressPhaseFinished {
			return
		}
		// Post-processors also run before the download (after_filter, before_dl).
		// Only those after the last format finished belong to this phase.
		if item.last.Phase != ProgressPhasePostProcessing && !item.allFinished() {
			return
		}
		t.emit(item, ev, ProgressPhasePostProcessing, nil)

	case "after_video":
		if item := t.items[key]; item != nil && item.last.Phase != ProgressPhaseFinished {
			t.emit(item, ev, ProgressPhaseFinished, nil)
		}
	}
}

// finish marks every video that is still open as finished. Called when yt-dlp
// exits successfully, in case after_video was never printed.
func (t *overallTracker) finish() {
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, item := range t.items {
		if item.emitted && item.last.Phase != ProgressPhaseFinished {
			t.emit(item, &overallEvent{}, ProgressPhaseFinished, nil)
		}
	}
}

func newOverallFormat(f overallEventFormat) *overallFormat {
	of := &overallFormat{id: f.FormatID}
	if f.Filesize != nil && *f.Filesize > 0 {
		of.declared, of.declaredExact = toInt64(f.Filesize), true
	} else if f.FilesizeApprox != nil {
		of.declared = toInt64(f.FilesizeApprox)
	}
	return of
}

func (item *overallItem) format(id string) *overallFormat {
	for _, f := range item.formats {
		if f.id == id {
			return f
		}
	}
	if id != "" && id == item.combinedID && len(item.formats) > 1 {
		// A merging downloader (ffmpeg) reports all formats as one file.
		combined := &overallFormat{id: id, declaredExact: true}
		for _, f := range item.formats {
			size, exact, _ := f.size()
			combined.declared += size
			combined.declaredExact = combined.declaredExact && exact
		}
		item.formats = []*overallFormat{combined}
		return combined
	}
	f := &overallFormat{id: id}
	item.formats = append(item.formats, f)
	return f
}

func (item *overallItem) allFinished() bool {
	return len(item.formats) > 0 && !slices.ContainsFunc(item.formats, func(f *overallFormat) bool { return !f.finished })
}

// emit computes the next snapshot from the item state and calls fn unless it
// equals the previous one. cur is the format that produced the event, if any.
func (t *overallTracker) emit(item *overallItem, ev *overallEvent, phase ProgressPhase, cur *overallFormat) {
	prev := item.last
	next := OverallProgress{
		ID:            prev.ID,
		PlaylistIndex: prev.PlaylistIndex,
		PlaylistCount: prev.PlaylistCount,
		Phase:         phase,
		FormatCount:   len(item.formats),
	}
	if !item.emitted {
		next.ID = ev.Info.ID
		next.PlaylistIndex = toInt(ev.Info.PlaylistIndex)
		next.PlaylistCount = toInt(ev.Info.NEntries)
	}

	var downloaded, total int64
	var fractions float64
	unknown, exact := false, true
	for i, f := range item.formats {
		size, sizeExact, known := f.size()
		downloaded += f.downloaded
		total += size
		exact = exact && sizeExact
		switch {
		case f.finished:
			fractions++
		case known && size > 0:
			fractions += min(1, float64(f.downloaded)/float64(size))
		default:
			unknown = true
		}
		if f == cur {
			next.FormatID, next.FormatIndex = f.id, i+1
		}
	}

	if !item.allFinished() {
		next.Downloaded = max(prev.Downloaded, downloaded)
		next.Total = max(total, next.Downloaded)
		next.TotalEstimated = !exact || unknown
		percent := 0.0
		switch {
		case unknown:
			// Some sizes are unknown: weigh formats equally instead of by bytes.
			percent = fractions / float64(len(item.formats)) * 100
		case next.Total > 0:
			percent = float64(next.Downloaded) / float64(next.Total) * 100
		}
		next.Percent = math.Min(100, math.Max(prev.Percent, percent))
	} else {
		// Every format is on disk: sizes are exact now, even if smaller than an
		// earlier estimate.
		next.Downloaded, next.Total, next.Percent = downloaded, downloaded, 100
	}
	if phase == ProgressPhaseFinished {
		// Without download events (the file already existed) the sizes stay as
		// declared; the video is complete regardless.
		next.Downloaded, next.Percent = next.Total, 100
	}

	if phase == ProgressPhaseDownloading && cur != nil && !cur.finished {
		next.Speed = toInt64(ev.Progress.Speed)
		next.ETA = time.Duration(toInt64(ev.Progress.ETA)) * time.Second
		if next.Speed > 0 && !next.TotalEstimated {
			next.ETA = time.Duration(float64(next.Total-next.Downloaded) / float64(next.Speed) * float64(time.Second)).Round(time.Second)
		}
	}
	if phase == ProgressPhasePostProcessing {
		next.PostProcessor = ev.Progress.PostProcessor
	}

	if item.emitted && next == prev {
		return
	}
	item.last, item.emitted = next, true
	t.fn(next)
}

func toInt64(v *float64) int64 {
	if v == nil || math.IsNaN(*v) || math.IsInf(*v, 0) || *v < 0 {
		return 0
	}
	return int64(*v)
}

func toInt(v *float64) int { return int(toInt64(v)) }
