//go:build !cgo

package web

import (
	"github.com/anthonynsimon/bild/imgio"
	"github.com/anthonynsimon/bild/transform"
	"os"
	"strings"

	"IB1/config"
)

func cleanImage(in string, out string) error {

	img, err := imgio.Open(in)
	if err != nil {
		return err
	}

	enc := imgio.PNGEncoder()
	ext := strings.Split(out, ".")
	if len(ext) > 0 && ext[len(ext)-1] != "png" {
		enc = imgio.JPEGEncoder(100)
	}
	return imgio.Save(out, img, enc)
}

func thumbnail(in string, out string) error {

	// fallback to ffmpeg if source is a gif image
	parts := strings.Split(in, ".")
	if len(parts) > 0 && parts[len(parts)-1] == "gif" {
		parts = strings.Split(out, "/")
		dst := config.Cfg.Media.Tmp + "/frame_" + parts[len(parts)-1]
		if err := extractFrame(in, dst); err != nil {
			return err
		}
		defer os.Remove(dst)
		in = dst
	}

	img, err := imgio.Open(in)
	if err != nil {
		return err
	}

	size := img.Bounds().Size()
	if err != nil {
		return err
	}
	w := size.X
	h := size.Y
	if w > h {
		h = h * 200 / w
		w = 200
	} else {
		w = w * 200 / h
		h = 200
	}

	img = transform.Resize(img, w, h, transform.Linear)
	return imgio.Save(out, img, imgio.PNGEncoder())
}
