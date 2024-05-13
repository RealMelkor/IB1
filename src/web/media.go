package web

import (
	"github.com/gin-gonic/gin"
	"github.com/h2non/bimg"
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
	"webm": false,
	"mp4": false,
}

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
	os.Remove(path)

	// rename to the sha256 hash of itself
	hash, err := sha256sum(out)
	if err != nil { return "", err }
	media := config.Cfg.Media.Path + "/" + hash + "." + extension
	err = os.Rename(out, media)
	if err != nil { return "", err }

	// create thumbnail
	err = thumbnail(media, config.Cfg.Media.Path +
				"/thumbnail/" + hash + ".png");
	if err != nil { return "", err }

	return hash + "." + extension, err
}
