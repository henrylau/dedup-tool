package videothumb

import (
	"errors"
	"fmt"
)

// ErrUnavailable is returned from EncodeJPEGThumbnail when the binary was built with
// CGO disabled — typical release cross-compile to windows (GOOS=windows CGO_ENABLED=0).
var ErrUnavailable = errors.New(
	`video thumbnails require CGO and FFmpeg development libraries compatible with github.com/asticode/go-astiav; this binary was built without them`,
)

// Options configures a single-frame thumbnail extraction.
type Options struct {
	SeekSeconds float64 // wall-clock seconds into the clip (approximate seek)
	MaxWidth    int     // upscale never applied; if > 0 and wider, scale down to this width
	Quality     int     // JPEG 1–100
	// NoSeek skips SeekFrame — read from BOF (HEIF/HIF stills).
	NoSeek bool
	// DebugLog receives thumbnail pipeline messages (caller may forward to UI).
	DebugLog func(msg string)
}

func (o Options) normalized() Options {
	if o.SeekSeconds < 0 {
		o.SeekSeconds = 0
	}
	if o.Quality < 1 {
		o.Quality = 1
	}
	if o.Quality > 100 {
		o.Quality = 100
	}
	return o
}

func (o Options) debugf(format string, args ...any) {
	if o.DebugLog == nil {
		return
	}
	o.DebugLog(fmt.Sprintf(format, args...))
}
