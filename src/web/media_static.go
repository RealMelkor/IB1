//go:build !cgo
package web

import (
	"github.com/gin-gonic/gin"
	"github.com/anthonynsimon/bild/imgio"
	"github.com/anthonynsimon/bild/transform"
	"mime/multipart"
	"crypto/rand"
	"crypto/sha256"
	"strings"
	"time"
	"fmt"
	"errors"
	"io"
	"os"
	"IB1/config"
)

func uniqueRandomName() (string, error) {
	now := time.Now().UnixMilli()
	bytes := make([]byte, 16)
	_, err := rand.Read(bytes)
	if err != nil { return "", err }
	str := fmt.Sprintf("%x", now)
	for _, v := range bytes {
		str += fmt.Sprintf("%x", v)
	}
	return str, nil
}

func sha256sum(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil { return "", err }
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil { return "", err }

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

var extensions = map[string]bool{
	"png": true,
	"jpg": true,
	"jpeg": true,
	"webp": true,
	"gif": true,
	"webm": false,
	"mp4": false,
}

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

func move(source string, destination string) error {
	src, err := os.Open(source)
	if err != nil { return err }
	defer src.Close()
	dst, err := os.Create(destination)
	if err != nil { return err }
	defer dst.Close()
	_, err = io.Copy(dst, src)
	if err != nil { return err }
	fi, err := os.Stat(source)
	if err != nil {
		os.Remove(destination)
		return err
	}
	err = os.Chmod(destination, fi.Mode())
	if err != nil {
		os.Remove(destination)
		return err
	}
	os.Remove(source)
	return nil
}

func uploadFile(c *gin.Context, file *multipart.FileHeader) (string, error) {

	// verify extension
	parts := strings.Split(file.Filename, ".")
	if len(parts) < 2 { return "", errors.New("no name extension") }
	extension := parts[len(parts) - 1]
	allowed, exist := extensions[extension]
	if !allowed || !exist {
		return "", errors.New("forbidden file extension")
	}

	// write file to disk
	name, err := uniqueRandomName()
	if err != nil { return "", err }
	path := config.Cfg.Media.Tmp + "/" + name + "." + extension
	if err = c.SaveUploadedFile(file, path); err != nil { return "", err }

	// clean up the metadata
	out := config.Cfg.Media.Tmp + "/clean_" + name + "." + extension
	if err := cleanImage(path, out); err != nil { return "", err }
	if extension == "gif" { os.Rename(path, out) } else { os.Remove(path) }

	// rename to the sha256 hash of itself
	hash, err := sha256sum(out)
	if err != nil { return "", err }
	media := config.Cfg.Media.Path + "/" + hash + "." + extension
	err = move(out, media)
	if err != nil { return "", err }

	// create thumbnail
	err = thumbnail(media, config.Cfg.Media.Path +
				"/thumbnail/" + hash + ".png");
	if err != nil { return "", err }

	return hash + "." + extension, err
}
