//go:build !cgo

package videothumb

// EncodeJPEGThumbnail is unavailable without CGO.
func EncodeJPEGThumbnail(_ string, o Options) ([]byte, error) {
	o = o.normalized()
	o.debugf("CGO disabled — thumbnail decoding not available in this binary")
	return nil, ErrUnavailable
}
