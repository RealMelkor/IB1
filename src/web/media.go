package web

import (
	"github.com/gin-gonic/gin"
	"mime/multipart"
	"crypto/rand"
	"crypto/sha256"
	"strings"
	"time"
	"fmt"
	"errors"
	"io"
	"os"
	"os/exec"
)

const mediaDir = "./media"
const thumbnailDir = "./thumbnail"
const tmpDir = "./tmp"

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
	"webm": true,
	"mp4": true,
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
	path := tmpDir + "/" + name + "." + extension
	if err = c.SaveUploadedFile(file, path); err != nil { return "", err }

	// clean up the metadata
	out := tmpDir + "/clean_" + name + "." + extension
	cmd := exec.Command("ffmpeg", "-i", path, out)
	if _, err := cmd.Output(); err != nil { return "", err }
	os.Remove(path)

	hash, err := sha256sum(out)
	if err != nil { return "", err }
	media := mediaDir + "/" + hash + "." + extension
	err = os.Rename(out, media)
	if err != nil { return "", err }

	cmd = exec.Command("ffmpegthumbnailer", "-i", media, "-o",
		thumbnailDir + "/" + hash + ".png")
	if _, err := cmd.Output(); err != nil { return "", err }

	return hash + "." + extension, err
}
