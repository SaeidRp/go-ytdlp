// Copyright (c) Liam Stanley <me@liamstanley.io>. All rights reserved. Use
// of this source code is governed by the MIT license that can be found in
// the LICENSE file.
//
// Code generated by cmd/codegen. DO NOT EDIT.
//
// Video Format Option Group

package ytdlp

// Video format code, see "FORMAT SELECTION" for more details
//
//   - See [UnsetFormat], for unsetting the flag.
//   - Format maps to cli flags: -f/--format=FORMAT.
func (c *Command) Format(format string) *Command {
	c.addFlag(&Flag{
		ID:   "format",
		Flag: "--format",
		Args: []string{format},
	})
	return c
}

// UnsetFormat unsets any flags that were previously set by
// [Format].
func (c *Command) UnsetFormat() *Command {
	c.removeFlagByID("format")
	return c
}

// Sort the formats by the fields given, see "Sorting Formats" for more details
//
//   - See [UnsetFormatSort], for unsetting the flag.
//   - FormatSort maps to cli flags: -S/--format-sort=SORTORDER.
func (c *Command) FormatSort(sortorder string) *Command {
	c.addFlag(&Flag{
		ID:   "format_sort",
		Flag: "--format-sort",
		Args: []string{sortorder},
	})
	return c
}

// UnsetFormatSort unsets any flags that were previously set by
// [FormatSort].
func (c *Command) UnsetFormatSort() *Command {
	c.removeFlagByID("format_sort")
	return c
}

// Force user specified sort order to have precedence over all fields, see "Sorting
// Formats" for more details (Alias: --S-force)
//
//   - See [UnsetFormatSortForce], for unsetting the flag.
//   - FormatSortForce maps to cli flags: --format-sort-force/--S-force=FORMAT.
func (c *Command) FormatSortForce() *Command {
	c.addFlag(&Flag{
		ID:   "format_sort_force",
		Flag: "--format-sort-force",
		Args: nil,
	})
	return c
}

// UnsetFormatSortForce unsets any flags that were previously set by
// [FormatSortForce].
func (c *Command) UnsetFormatSortForce() *Command {
	c.removeFlagByID("format_sort_force")
	return c
}

// Some fields have precedence over the user specified sort order (default)
//
//   - See [UnsetNoFormatSortForce], for unsetting the flag.
//   - NoFormatSortForce maps to cli flags: --no-format-sort-force=FORMAT.
func (c *Command) NoFormatSortForce() *Command {
	c.addFlag(&Flag{
		ID:   "format_sort_force",
		Flag: "--no-format-sort-force",
		Args: nil,
	})
	return c
}

// UnsetNoFormatSortForce unsets any flags that were previously set by
// [NoFormatSortForce].
func (c *Command) UnsetNoFormatSortForce() *Command {
	c.removeFlagByID("format_sort_force")
	return c
}

// Allow multiple video streams to be merged into a single file
//
//   - See [UnsetVideoMultistreams], for unsetting the flag.
//   - VideoMultistreams maps to cli flags: --video-multistreams.
func (c *Command) VideoMultistreams() *Command {
	c.addFlag(&Flag{
		ID:   "allow_multiple_video_streams",
		Flag: "--video-multistreams",
		Args: nil,
	})
	return c
}

// UnsetVideoMultistreams unsets any flags that were previously set by
// [VideoMultistreams].
func (c *Command) UnsetVideoMultistreams() *Command {
	c.removeFlagByID("allow_multiple_video_streams")
	return c
}

// Only one video stream is downloaded for each output file (default)
//
//   - See [UnsetNoVideoMultistreams], for unsetting the flag.
//   - NoVideoMultistreams maps to cli flags: --no-video-multistreams.
func (c *Command) NoVideoMultistreams() *Command {
	c.addFlag(&Flag{
		ID:   "allow_multiple_video_streams",
		Flag: "--no-video-multistreams",
		Args: nil,
	})
	return c
}

// UnsetNoVideoMultistreams unsets any flags that were previously set by
// [NoVideoMultistreams].
func (c *Command) UnsetNoVideoMultistreams() *Command {
	c.removeFlagByID("allow_multiple_video_streams")
	return c
}

// Allow multiple audio streams to be merged into a single file
//
//   - See [UnsetAudioMultistreams], for unsetting the flag.
//   - AudioMultistreams maps to cli flags: --audio-multistreams.
func (c *Command) AudioMultistreams() *Command {
	c.addFlag(&Flag{
		ID:   "allow_multiple_audio_streams",
		Flag: "--audio-multistreams",
		Args: nil,
	})
	return c
}

// UnsetAudioMultistreams unsets any flags that were previously set by
// [AudioMultistreams].
func (c *Command) UnsetAudioMultistreams() *Command {
	c.removeFlagByID("allow_multiple_audio_streams")
	return c
}

// Only one audio stream is downloaded for each output file (default)
//
//   - See [UnsetNoAudioMultistreams], for unsetting the flag.
//   - NoAudioMultistreams maps to cli flags: --no-audio-multistreams.
func (c *Command) NoAudioMultistreams() *Command {
	c.addFlag(&Flag{
		ID:   "allow_multiple_audio_streams",
		Flag: "--no-audio-multistreams",
		Args: nil,
	})
	return c
}

// UnsetNoAudioMultistreams unsets any flags that were previously set by
// [NoAudioMultistreams].
func (c *Command) UnsetNoAudioMultistreams() *Command {
	c.removeFlagByID("allow_multiple_audio_streams")
	return c
}

// AllFormats sets the "all-formats" flag (no description specified).
//
//   - See [UnsetAllFormats], for unsetting the flag.
//   - AllFormats maps to cli flags: --all-formats (hidden).
func (c *Command) AllFormats() *Command {
	c.addFlag(&Flag{
		ID:   "format",
		Flag: "--all-formats",
		Args: nil,
	})
	return c
}

// UnsetAllFormats unsets any flags that were previously set by
// [AllFormats].
func (c *Command) UnsetAllFormats() *Command {
	c.removeFlagByID("format")
	return c
}

// Prefer video formats with free containers over non-free ones of same quality.
// Use with "-S ext" to strictly prefer free containers irrespective of quality
//
//   - See [UnsetPreferFreeFormats], for unsetting the flag.
//   - PreferFreeFormats maps to cli flags: --prefer-free-formats.
func (c *Command) PreferFreeFormats() *Command {
	c.addFlag(&Flag{
		ID:   "prefer_free_formats",
		Flag: "--prefer-free-formats",
		Args: nil,
	})
	return c
}

// UnsetPreferFreeFormats unsets any flags that were previously set by
// [PreferFreeFormats].
func (c *Command) UnsetPreferFreeFormats() *Command {
	c.removeFlagByID("prefer_free_formats")
	return c
}

// Don't give any special preference to free containers (default)
//
//   - See [UnsetNoPreferFreeFormats], for unsetting the flag.
//   - NoPreferFreeFormats maps to cli flags: --no-prefer-free-formats.
func (c *Command) NoPreferFreeFormats() *Command {
	c.addFlag(&Flag{
		ID:   "prefer_free_formats",
		Flag: "--no-prefer-free-formats",
		Args: nil,
	})
	return c
}

// UnsetNoPreferFreeFormats unsets any flags that were previously set by
// [NoPreferFreeFormats].
func (c *Command) UnsetNoPreferFreeFormats() *Command {
	c.removeFlagByID("prefer_free_formats")
	return c
}

// Make sure formats are selected only from those that are actually downloadable
//
//   - See [UnsetCheckFormats], for unsetting the flag.
//   - CheckFormats maps to cli flags: --check-formats.
func (c *Command) CheckFormats() *Command {
	c.addFlag(&Flag{
		ID:   "check_formats",
		Flag: "--check-formats",
		Args: nil,
	})
	return c
}

// UnsetCheckFormats unsets any flags that were previously set by
// [CheckFormats].
func (c *Command) UnsetCheckFormats() *Command {
	c.removeFlagByID("check_formats")
	return c
}

// Check all formats for whether they are actually downloadable
//
//   - See [UnsetCheckAllFormats], for unsetting the flag.
//   - CheckAllFormats maps to cli flags: --check-all-formats.
func (c *Command) CheckAllFormats() *Command {
	c.addFlag(&Flag{
		ID:   "check_formats",
		Flag: "--check-all-formats",
		Args: nil,
	})
	return c
}

// UnsetCheckAllFormats unsets any flags that were previously set by
// [CheckAllFormats].
func (c *Command) UnsetCheckAllFormats() *Command {
	c.removeFlagByID("check_formats")
	return c
}

// Do not check that the formats are actually downloadable
//
//   - See [UnsetNoCheckFormats], for unsetting the flag.
//   - NoCheckFormats maps to cli flags: --no-check-formats.
func (c *Command) NoCheckFormats() *Command {
	c.addFlag(&Flag{
		ID:   "check_formats",
		Flag: "--no-check-formats",
		Args: nil,
	})
	return c
}

// UnsetNoCheckFormats unsets any flags that were previously set by
// [NoCheckFormats].
func (c *Command) UnsetNoCheckFormats() *Command {
	c.removeFlagByID("check_formats")
	return c
}

// List available formats of each video. Simulate unless --no-simulate is used
//
//   - See [UnsetListFormats], for unsetting the flag.
//   - ListFormats maps to cli flags: -F/--list-formats.
func (c *Command) ListFormats() *Command {
	c.addFlag(&Flag{
		ID:   "listformats",
		Flag: "--list-formats",
		Args: nil,
	})
	return c
}

// UnsetListFormats unsets any flags that were previously set by
// [ListFormats].
func (c *Command) UnsetListFormats() *Command {
	c.removeFlagByID("listformats")
	return c
}

// ListFormatsAsTable sets the "list-formats-as-table" flag (no description specified).
//
//   - See [UnsetListFormatsAsTable], for unsetting the flag.
//   - ListFormatsAsTable maps to cli flags: --list-formats-as-table (hidden).
func (c *Command) ListFormatsAsTable() *Command {
	c.addFlag(&Flag{
		ID:   "listformats_table",
		Flag: "--list-formats-as-table",
		Args: nil,
	})
	return c
}

// UnsetListFormatsAsTable unsets any flags that were previously set by
// [ListFormatsAsTable].
func (c *Command) UnsetListFormatsAsTable() *Command {
	c.removeFlagByID("listformats_table")
	return c
}

// ListFormatsOld sets the "list-formats-old" flag (no description specified).
//
//   - See [UnsetListFormatsOld], for unsetting the flag.
//   - ListFormatsOld maps to cli flags: --list-formats-old/--no-list-formats-as-table (hidden).
func (c *Command) ListFormatsOld() *Command {
	c.addFlag(&Flag{
		ID:   "listformats_table",
		Flag: "--list-formats-old",
		Args: nil,
	})
	return c
}

// UnsetListFormatsOld unsets any flags that were previously set by
// [ListFormatsOld].
func (c *Command) UnsetListFormatsOld() *Command {
	c.removeFlagByID("listformats_table")
	return c
}

// Containers that may be used when merging formats, separated by "/", e.g.
// "mp4/mkv". Ignored if no merge is required. (currently supported: avi, flv, mkv,
// mov, mp4, webm)
//
//   - See [UnsetMergeOutputFormat], for unsetting the flag.
//   - MergeOutputFormat maps to cli flags: --merge-output-format=FORMAT.
func (c *Command) MergeOutputFormat(format string) *Command {
	c.addFlag(&Flag{
		ID:   "merge_output_format",
		Flag: "--merge-output-format",
		Args: []string{format},
	})
	return c
}

// UnsetMergeOutputFormat unsets any flags that were previously set by
// [MergeOutputFormat].
func (c *Command) UnsetMergeOutputFormat() *Command {
	c.removeFlagByID("merge_output_format")
	return c
}

// AllowUnplayableFormats sets the "allow-unplayable-formats" flag (no description specified).
//
//   - See [UnsetAllowUnplayableFormats], for unsetting the flag.
//   - AllowUnplayableFormats maps to cli flags: --allow-unplayable-formats (hidden).
func (c *Command) AllowUnplayableFormats() *Command {
	c.addFlag(&Flag{
		ID:   "allow_unplayable_formats",
		Flag: "--allow-unplayable-formats",
		Args: nil,
	})
	return c
}

// UnsetAllowUnplayableFormats unsets any flags that were previously set by
// [AllowUnplayableFormats].
func (c *Command) UnsetAllowUnplayableFormats() *Command {
	c.removeFlagByID("allow_unplayable_formats")
	return c
}

// NoAllowUnplayableFormats sets the "no-allow-unplayable-formats" flag (no description specified).
//
//   - See [UnsetNoAllowUnplayableFormats], for unsetting the flag.
//   - NoAllowUnplayableFormats maps to cli flags: --no-allow-unplayable-formats (hidden).
func (c *Command) NoAllowUnplayableFormats() *Command {
	c.addFlag(&Flag{
		ID:   "allow_unplayable_formats",
		Flag: "--no-allow-unplayable-formats",
		Args: nil,
	})
	return c
}

// UnsetNoAllowUnplayableFormats unsets any flags that were previously set by
// [NoAllowUnplayableFormats].
func (c *Command) UnsetNoAllowUnplayableFormats() *Command {
	c.removeFlagByID("allow_unplayable_formats")
	return c
}
