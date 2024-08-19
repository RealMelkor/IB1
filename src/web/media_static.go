//go:build !cgo
package web

import (
	"github.com/anthonynsimon/bild/imgio"
	"github.com/anthonynsimon/bild/transform"
	"strings"
)

func cleanImage(in string, out string) error {

	img, err := imgio.Open(in)
	if err != nil { return err }

	enc := imgio.PNGEncoder()
	ext := strings.Split(out, ".")
	if len(ext) > 0 && ext[len(ext) - 1] != "png" {
		enc = imgio.JPEGEncoder(100)
	}
	return imgio.Save(out, img, enc)
}

func thumbnail(in string, out string) error {

	img, err := imgio.Open(in)
	if err != nil { return err }

	size := img.Bounds().Size()
	if err != nil { return err }
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
