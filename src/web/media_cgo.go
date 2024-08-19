//go:build cgo
package web

import (
	"github.com/h2non/bimg"
)

func cleanImage(in string, out string) error {

	buffer, err := bimg.Read(in)
	if err != nil { return err }

	img, err := bimg.NewImage(buffer).Process(
			bimg.Options{StripMetadata: true})
	if err != nil { return err }

	bimg.Write(out, img)
	return nil
}

func thumbnail(in string, out string) error {

	buffer, err := bimg.Read(in)
	if err != nil { return err }

	img := bimg.NewImage(buffer)

	size, err := img.Size()
	if err != nil { return err }
	w := size.Width
	h := size.Height
	if w > h {
		h = h * 200 / w
		w = 200
	} else {
		w = w * 200 / h
		h = 200
	}

	newImage, err := img.Resize(w, h)
	if err != nil { return err }

	return bimg.Write(out, newImage)
}
