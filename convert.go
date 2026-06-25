package media

import (
	"bytes"

	"github.com/disintegration/imaging"
)

// generateConversion decodes an image and applies a conversion, returning JPEG
// bytes. Fit preserves aspect ratio within WxH; otherwise the image is cropped
// to fill an exact WxH thumbnail.
func generateConversion(data []byte, c Conversion) ([]byte, error) {
	img, err := imaging.Decode(bytes.NewReader(data), imaging.AutoOrientation(true))
	if err != nil {
		return nil, err
	}
	var out = img
	if c.Fit {
		out = imaging.Fit(img, c.Width, c.Height, imaging.Lanczos)
	} else {
		out = imaging.Thumbnail(img, c.Width, c.Height, imaging.Lanczos)
	}
	var buf bytes.Buffer
	if err := imaging.Encode(&buf, out, imaging.JPEG, imaging.JPEGQuality(85)); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
