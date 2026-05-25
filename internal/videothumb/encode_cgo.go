//go:build cgo && videothumb

package videothumb

import (
	"bytes"
	"errors"
	"fmt"
	"image/jpeg"
	"math"
	"path/filepath"
	"strings"
	"time"

	"github.com/asticode/go-astiav"
)

// EncodeJPEGThumbnail decodes one video frame near SeekSeconds and returns JPEG bytes.
func EncodeJPEGThumbnail(videoPath string, o Options) ([]byte, error) {
	o = o.normalized()
	tAll := time.Now()
	astiav.SetLogLevel(astiav.LogLevelFatal)

	baseName := filepath.Base(videoPath)
	o.debugf("start file=%s seek=%gs noSeek=%v maxW=%d q=%d", baseName, o.SeekSeconds, o.NoSeek, o.MaxWidth, o.Quality)

	absVideoPath, err := filepath.Abs(videoPath)
	if err != nil {
		o.debugf("failed abs path: %v", err)
		return nil, fmt.Errorf("resolve path: %w", err)
	}

	pkt := astiav.AllocPacket()
	defer pkt.Free()

	frame := astiav.AllocFrame()
	defer frame.Free()

	dstFrame := astiav.AllocFrame()
	defer dstFrame.Free()

	fc := astiav.AllocFormatContext()
	if fc == nil {
		o.debugf("alloc format context returned nil")
		return nil, errors.New("alloc format context: nil")
	}
	defer fc.Free()

	t0 := time.Now()
	if err := fc.OpenInput(absVideoPath, nil, nil); err != nil {
		o.debugf("OpenInput failed: %v", err)
		return nil, fmt.Errorf("open input: %w", err)
	}
	o.debugf("OpenInput ok (%s)", time.Since(t0))
	defer fc.CloseInput()

	t0 = time.Now()
	if err := fc.FindStreamInfo(nil); err != nil {
		o.debugf("FindStreamInfo failed: %v", err)
		return nil, fmt.Errorf("find stream info: %w", err)
	}
	o.debugf("FindStreamInfo ok, nb_streams=%d (%s)", fc.NbStreams(), time.Since(t0))

	videoStream, decCodec, err := fc.FindBestStream(astiav.MediaTypeVideo, -1, -1)
	if err != nil && o.NoSeek {
		for _, stm := range fc.Streams() {
			cp := stm.CodecParameters()
			if cp == nil || cp.MediaType() != astiav.MediaTypeVideo {
				continue
			}
			candidate := astiav.FindDecoder(cp.CodecID())
			if candidate == nil {
				continue
			}
			videoStream, decCodec, err = stm, candidate, nil
			o.debugf("picked video-like stream manually index=%d", stm.Index())
			break
		}
	}
	if err != nil {
		o.debugf("FindBestStream video failed: %v", err)
		return nil, fmt.Errorf("no video stream: %w", err)
	}
	tbn := videoStream.TimeBase().Num()
	tbd := videoStream.TimeBase().Den()
	dcName := "?"
	if decCodec != nil {
		dcName = decCodec.Name()
	}
	o.debugf("video stream index=%d codec=%s time_base=%d/%d", videoStream.Index(), dcName, tbn, tbd)

	decCC := astiav.AllocCodecContext(decCodec)
	if decCC == nil {
		o.debugf("AllocCodecContext nil")
		return nil, errors.New("alloc decoder context: nil")
	}
	defer decCC.Free()

	if err := videoStream.CodecParameters().ToCodecContext(decCC); err != nil {
		o.debugf("ToCodecContext failed: %v", err)
		return nil, fmt.Errorf("copy codec params: %w", err)
	}
	if err := decCC.Open(decCodec, nil); err != nil {
		o.debugf("decoder Open failed: %v", err)
		return nil, fmt.Errorf("open decoder: %w", err)
	}

	if !o.NoSeek {
		us := int64(math.Round(o.SeekSeconds * 1e6))
		seekTS := astiav.RescaleQRnd(
			us,
			astiav.NewRational(1, 1000000),
			videoStream.TimeBase(),
			astiav.RoundingNearInf,
		)
		flags := astiav.NewSeekFlags(astiav.SeekFlagBackward)
		o.debugf("SeekFrame ts=%d (stream units)", seekTS)
		tSeek := time.Now()
		if err := fc.SeekFrame(videoStream.Index(), seekTS, flags); err != nil {
			o.debugf("SeekFrame failed: %v", err)
			return nil, fmt.Errorf("seek: %w", err)
		}
		o.debugf("SeekFrame ok (%s)", time.Since(tSeek))
	} else {
		o.debugf("skipping SeekFrame (still image / HEIF)")
	}

	pktReads := 0
	for {
		if err := fc.ReadFrame(pkt); err != nil {
			if errors.Is(err, astiav.ErrEof) {
				o.debugf("eof after %d packets read demux", pktReads)
				return nil, errors.New("eof before a video frame was decoded")
			}
			o.debugf("ReadFrame error: %v", err)
			return nil, fmt.Errorf("read frame: %w", err)
		}
		pktReads++

		if pkt.StreamIndex() != videoStream.Index() {
			pkt.Unref()
			continue
		}

		if err := decCC.SendPacket(pkt); err != nil {
			pkt.Unref()
			o.debugf("SendPacket failed: %v", err)
			return nil, fmt.Errorf("send packet: %w", err)
		}
		pkt.Unref()

		for {
			if err := decCC.ReceiveFrame(frame); err != nil {
				if errors.Is(err, astiav.ErrEagain) {
					break
				}
				if errors.Is(err, astiav.ErrEof) {
					o.debugf("ReceiveFrame eof (decoder flushed)")
					return nil, errors.New("decoder flushed before thumbnail")
				}
				o.debugf("ReceiveFrame error: %v", err)
				return nil, fmt.Errorf("receive frame: %w", err)
			}

			px := strings.TrimSpace(frame.PixelFormat().String())
			o.debugf("decoded frame %dx%d pts=%d fmt=%s", frame.Width(), frame.Height(), frame.Pts(), px)

			dstW := frame.Width()
			dstH := frame.Height()
			if o.MaxWidth > 0 && frame.Width() > o.MaxWidth {
				dstW = o.MaxWidth
				dstH = int(math.Round(float64(frame.Height()) * float64(o.MaxWidth) / float64(frame.Width())))
			}
			o.debugf("scale to %dx%d", dstW, dstH)

			tScale := time.Now()
			sws, err := astiav.CreateSoftwareScaleContext(
				frame.Width(),
				frame.Height(),
				frame.PixelFormat(),
				dstW,
				dstH,
				astiav.PixelFormatRgba,
				astiav.NewSoftwareScaleContextFlags(astiav.SoftwareScaleContextFlagBilinear),
			)
			if err != nil {
				frame.Unref()
				o.debugf("CreateSoftwareScaleContext failed: %v", err)
				return nil, fmt.Errorf("software scale context: %w", err)
			}

			if err := sws.ScaleFrame(frame, dstFrame); err != nil {
				sws.Free()
				frame.Unref()
				o.debugf("ScaleFrame failed: %v", err)
				return nil, fmt.Errorf("scale: %w", err)
			}
			sws.Free()
			frame.Unref()
			o.debugf("sws ok (%s)", time.Since(tScale))

			img, err := dstFrame.Data().GuessImageFormat()
			if err != nil {
				o.debugf("GuessImageFormat failed: %v", err)
				return nil, fmt.Errorf("guess image format: %w", err)
			}
			if err := dstFrame.Data().ToImage(img); err != nil {
				o.debugf("ToImage failed: %v", err)
				return nil, fmt.Errorf("frame to image: %w", err)
			}

			var buf bytes.Buffer
			if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: o.Quality}); err != nil {
				o.debugf("jpeg encode failed: %v", err)
				return nil, fmt.Errorf("encode jpeg: %w", err)
			}
			b := buf.Bytes()
			o.debugf("done jpeg_bytes=%d total_elapsed=%s", len(b), time.Since(tAll))
			return b, nil
		}
	}
}
