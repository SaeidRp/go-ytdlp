package ytdlp

import (
	"bytes"
	"fmt"

	"github.com/saeidrp/go-ytdlp/template"
)

// progressLinePrefix is used to prefix the progress line, making it easier to identify during
// output parsing.
const progressLinePrefix = "dl:"

// DownloadProgressFunc is a callback function that is called whenever there is a progress update.
//
// Deprecated: use [Command.OverallProgressFunc] and [OverallProgress] instead.
type DownloadProgressFunc = func(DownloadProgress)

// DownloadStatus represents the status of a download.
type DownloadStatus string

const (
	// StatusPreProcessing represents the download status when the download is pre-processing.
	StatusStarting DownloadStatus = "starting"
	// StatusDownloading represents the download status when the download is in progress.
	StatusDownloading DownloadStatus = "downloading"
	// StatusPostProcessing represents the download status when the download is post-processing.
	StatusPostProcessing DownloadStatus = "post_processing"
	// StatusFinished represents the download status when the download is finished.
	StatusFinished DownloadStatus = "finished"
)

// DownloadProgress represents the progress of a download.
//
// Deprecated: use [OverallProgress] instead. DownloadProgress is per file and
// restarts for every requested format, never reaches the audio stream of a
// "bv+ba" download, and drops float sizes (fragmented HLS/DASH estimates).
type DownloadProgress struct {
	// ID represents the video ID.
	ID string `ytdlp:"info.id"`
	// PlaylistID represents the playlist id if the download is part of a playlist.
	PlaylistID string `ytdlp:"info.playlist_id"`
	// Title represents the title of the video.
	Title string `ytdlp:"info.title"`
	// TotalBytes represents the size of the video in bytes.
	// if the ytdlp total_bytes doesn't exist, it will then fallback to the total_bytes_estimate.
	TotalBytes int64 `ytdlp:"progress.total_bytes"`
	// TotalBytesEstimated represents the estimated size of the video in bytes.
	TotalBytesEstimated int64 `ytdlp:"progress.total_bytes_estimate"`
	// DownloadedBytes represents the number of bytes downloaded.
	DownloadedBytes int64 `ytdlp:"progress.downloaded_bytes"`
	// Speed represents the download speed in bytes per second.
	Speed int64 `ytdlp:"progress.speed"`
	// Percent represents the download progress as a percentage.
	Percent float64 `ytdlp:"progress._percent_str,formatter=percentToNumber"`
	// Status represents the status of the download.
	Status DownloadStatus `ytdlp:"progress.status"`
	// PlaylistCount represents the total number of videos in the playlist.
	PlaylistCount int `ytdlp:"info.playlist_count"`
	// PlaylistIndex represents the index of the video in the playlist.
	PlaylistIndex int `ytdlp:"info.playlist_index"`
}

// IsPlaylist returns true if the download is part of a playlist.
func (p *DownloadProgress) IsPlaylist() bool {
	return p.PlaylistCount > 0
}

// GetDownloadProgressTemplate returns the template for the ytdlp's download progress event.
// The template will be used by ytdlp to send a progress event during the download.
func GetDownloadProgressTemplate() (string, error) {
	var downloadProgress DownloadProgress
	templ, err := template.MarshalTemplate(downloadProgress)
	if err != nil {
		return "", fmt.Errorf("failed to marshal template: %w", err)
	}

	return progressLinePrefix + string(templ), nil
}

// progressStatus represents the status of a ytdlp download progress event.
type progressStatus string

const (
	progressPreProcessing      progressStatus = "pre_processing"
	progressStarting           progressStatus = "starting"
	progressDownloading        progressStatus = "downloading"
	progressFinished           progressStatus = "finished"
	progressPostProcessing     progressStatus = "post_processing"
	progressVideoDownloaded    progressStatus = "video_downloaded"
	progressPlaylistDownloaded progressStatus = "playlist_downloaded"
)

// progressEvent represents a download progress event.
type progressEvent struct {
	ID         string `ytdlp:"id"`
	PlaylistID string `ytdlp:"playlist_id"`
	Title      string `ytdlp:"title"`
	Status     progressStatus
}

// GetProgressPreProcessTemplate returns the template for the ytdlp's pre_process event.
func GetProgressPreProcessTemplate() (string, error) {
	event := progressEvent{Status: progressPreProcessing}
	templ, err := template.MarshalTemplate(event)
	if err != nil {
		return "", fmt.Errorf("failed to marshal template: %w", err)
	}

	return fmt.Sprintf("pre_process:%s%s", progressLinePrefix, string(templ)), nil
}

// GetProgressBeforeDownloadTemplate returns the template for the ytdlp's before_dl event.
// The template will be used by ytdlp to send a progress event before starting the download.
func GetProgressBeforeDownloadTemplate() (string, error) {
	event := progressEvent{Status: progressStarting}
	templ, err := template.MarshalTemplate(event)
	if err != nil {
		return "", fmt.Errorf("failed to marshal template: %w", err)
	}

	return fmt.Sprintf("before_dl:%s%s", progressLinePrefix, string(templ)), nil
}

// GetProgressPostProcessTemplate returns the template for the ytdlp's post_process event.
// The template will be used by ytdlp to send a progress event when the video is post-processing.
func GetProgressPostProcessTemplate() (string, error) {
	event := struct {
		ID         string `ytdlp:"id"`
		PlaylistID string `ytdlp:"playlist_id"`
		Status     progressStatus
	}{Status: progressPostProcessing}
	templ, err := template.MarshalTemplate(event)
	if err != nil {
		return "", fmt.Errorf("failed to marshal template: %w", err)
	}

	return fmt.Sprintf("post_process:%s%s", progressLinePrefix, string(templ)), nil
}

// GetProgressVideoDownloadedTemplate returns the template for the ytdlp's after_video event.
// The template will be used by ytdlp to send a progress event when the video is being downloaded.
func GetProgressVideoDownloadedTemplate() (string, error) {
	event := struct {
		ID         string `ytdlp:"id"`
		PlaylistID string `ytdlp:"playlist_id"`
		Status     progressStatus
	}{Status: progressVideoDownloaded}
	templ, err := template.MarshalTemplate(event)
	if err != nil {
		return "", fmt.Errorf("failed to marshal template: %w", err)
	}

	return fmt.Sprintf("after_video:%s%s", progressLinePrefix, string(templ)), nil
}

// GetProgressPlaylistDownloadedTemplate returns the template for the ytdlp's playlist event.
// The template will be used by ytdlp to send a progress event when the playlist is being downloaded.
func GetProgressPlaylistDownloadedTemplate() (string, error) {
	event := struct {
		ID     string `ytdlp:"id"`
		Status progressStatus
	}{Status: progressPlaylistDownloaded}
	templ, err := template.MarshalTemplate(event)
	if err != nil {
		return "", fmt.Errorf("failed to marshal template: %w", err)
	}

	return fmt.Sprintf("playlist:%s%s", progressLinePrefix, string(templ)), nil
}

// SetProgressFn sets the progress function to be called on each progress event.
//
// Deprecated: use [Command.OverallProgressFunc], which reports one monotonic
// progress across all requested formats plus post-processing. SetProgressFn
// is kept for existing callers; it injects --progress, --newline,
// --progress-template and several --print templates (which makes yt-dlp
// quiet). Setting it clears [Command.ProgressFunc] and
// [Command.OverallProgressFunc].
func (c *Command) SetProgressFn(progressFunc DownloadProgressFunc) *Command {
	c.clearProgressHandlers()
	if progressFunc == nil {
		return c
	}

	// Template marshaling of these fixed structs cannot fail.
	downloadTempl, _ := GetDownloadProgressTemplate()
	beforeDownloadTempl, _ := GetProgressBeforeDownloadTemplate()
	postProcessTempl, _ := GetProgressPostProcessTemplate()
	videoDownloadedTempl, _ := GetProgressVideoDownloadedTemplate()

	// --newline makes yt-dlp print each progress update on its own line instead
	// of overwriting it with a carriage return.
	c.Progress().ProgressTemplate(downloadTempl).Newline().
		Print(beforeDownloadTempl).
		Print(postProcessTempl).
		Print(videoDownloadedTempl)

	c.mu.Lock()
	c.legacyProgress = progressFunc
	c.mu.Unlock()
	return c
}

// legacyProgress turns the [SetProgressFn] template lines into
// [DownloadProgress] callbacks. It only reads stdout.
type legacyProgress struct {
	fn        DownloadProgressFunc
	downloads map[string]*DownloadProgress
}

func newLegacyProgress(fn DownloadProgressFunc) *legacyProgress {
	return &legacyProgress{fn: fn, downloads: make(map[string]*DownloadProgress)}
}

func (l *legacyProgress) consume(line []byte) bool {
	if !bytes.HasPrefix(line, []byte(progressLinePrefix)) {
		return false
	}
	l.handle(bytes.TrimPrefix(line, []byte(progressLinePrefix)))
	return true
}

// handle is the original fork implementation, unchanged.
func (l *legacyProgress) handle(line []byte) {
	var event progressEvent
	if err := template.Unmarshal(line, &event); err != nil {
		return
	}

	progress, ok := l.downloads[event.ID]
	if !ok {
		progress = &DownloadProgress{}
		l.downloads[event.ID] = progress
	}

	switch event.Status {
	case progressStarting:
		progress.ID = event.ID
		progress.PlaylistID = event.PlaylistID
		progress.Title = event.Title
		progress.Status = StatusStarting
	case progressDownloading:
		// After finishing the download of the video file, yt-dlp may proceed to download
		// additional components such as audio tracks, subtitles, etc., if required.
		// In this situation, it is necessary to ignore any further progress updates with
		// the status of `downloading` once the progress has been marked as `downloadStatusPostProcessing`.
		// This ensures that the progress is not overwritten by subsequent
		// download updates of additional files.
		if progress.Status == StatusPostProcessing {
			return
		}

		if err := template.Unmarshal(line, progress); err != nil {
			return
		}
	case progressFinished, progressPostProcessing:
		progress.Status = StatusPostProcessing
	case progressVideoDownloaded:
		progress.Status = StatusFinished
	default:
		// Prevent triggering the progressFunc if the status is unknown.
		return
	}

	l.fn(*progress)
}

func (l *legacyProgress) list() []*DownloadProgress {
	downloads := make([]*DownloadProgress, 0, len(l.downloads))
	for _, d := range l.downloads {
		downloads = append(downloads, d)
	}
	return downloads
}
