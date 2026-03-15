package media

import (
	"github.com/corona10/goimagehash"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"

	"IB1/db"
)

func isImageBanned(r io.Reader) error {
	img, _, err := image.Decode(r)
	if err != nil {
		return err
	}
	v, err := goimagehash.AverageHash(img)
	if err != nil {
		return err
	}
	return db.IsImageBanned(*v)
}

func Ban(hash string) error {
	r, err := mediaReader(hash)
	if err != nil {
		return err
	}
	img, _, err := image.Decode(r)
	if err != nil {
		return err
	}
	v, err := goimagehash.AverageHash(img)
	if err != nil {
		return err
	}
	return db.BanImage(*v)
}
