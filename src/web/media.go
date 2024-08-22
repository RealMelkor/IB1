package web

import (
	"os"
	"io"
	"time"
	"fmt"
	"strings"
	"errors"
	"crypto/rand"
	"crypto/sha256"
	"mime/multipart"

	"github.com/gabriel-vasile/mimetype"
	"github.com/gin-gonic/gin"
	"IB1/config"
	"IB1/db"
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

func uploadFile(c *gin.Context, file *multipart.FileHeader) (string, error) {

	if uint64(file.Size) > config.Cfg.Media.MaxSize {
		return "", errors.New("media is above size limit")
	}

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
	if config.Cfg.Media.InDatabase { // store media in database
		tn := config.Cfg.Media.Tmp + "/thumbnail_" + hash + ".png"
		if err := thumbnail(out, tn); err != nil { return "", err }
		defer os.Remove(tn)
		defer os.Remove(out)
		tn_data, err := os.ReadFile(tn)
		if err != nil { return "", err }
		data, err := os.ReadFile(out)
		if err != nil { return "", err }
		mime := mimetype.Detect(data).String()
		err = db.AddMedia(data, tn_data, hash, mime)
		if err != nil { return "", err }
		return hash + "." + extension, nil
	}
	media := config.Cfg.Media.Path + "/" + hash + "." + extension
	err = move(out, media)
	if err != nil { return "", err }

	// create thumbnail
	err = thumbnail(media, config.Cfg.Media.Path +
				"/thumbnail/" + hash + ".png");
	if err != nil { return "", err }

	return hash + "." + extension, nil
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
