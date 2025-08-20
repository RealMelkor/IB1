package web

import (
	"os"
	"os/exec"
	"io"
	"time"
	"fmt"
	"bytes"
	"errors"
	"crypto/rand"
	"golang.org/x/crypto/blake2b"
	"mime/multipart"
	"strings"

	"github.com/gabriel-vasile/mimetype"

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

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil { return "", err }
	defer f.Close()

	h, err := blake2b.New256(config.Cfg.Media.Key)
	if err != nil { return "", err }
	if _, err := io.Copy(h, f); err != nil { return "", err }

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

var extensions = map[string]db.MediaType{
	".png": db.MEDIA_PICTURE,
	".jpg": db.MEDIA_PICTURE,
	".jpeg": db.MEDIA_PICTURE,
	".gif": db.MEDIA_PICTURE,
	".webp": db.MEDIA_PICTURE,
	".webm": db.MEDIA_VIDEO,
	".mp4": db.MEDIA_VIDEO,
}

func saveUploadedFile(file *multipart.FileHeader, out string) error {
	src, err := file.Open()
	if err != nil { return err }
	defer src.Close()

	dst, err := os.Create(out)
	if err != nil { return err }
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}

func validExtension(extension string) (db.MediaType, error) {
	mediaType, exist := extensions[extension]
	if mediaType == db.MEDIA_VIDEO && !config.Cfg.Media.AllowVideos {
		return 0, errors.New("video support not enabled")
	}
	if !exist {
		return 0, errors.New("forbidden file extension")
	}
	return mediaType, nil
}

func uploadFile(file *multipart.FileHeader,
		approved bool, spoiler bool) (string, error) {

	if uint64(file.Size) > config.Cfg.Media.MaxSize {
		return "", errors.New("media is above size limit")
	}

	// write file to disk
	name, err := uniqueRandomName()
	if err != nil { return "", err }
	path := config.Cfg.Media.Tmp + "/" + name
	if err = saveUploadedFile(file, path); err != nil { return "", err }
	defer os.Remove(path)

	// verify extension
	mime, err := mimetype.DetectFile(path)
	if err != nil { return "", err }
	extension := mime.Extension()
	mediaType, err := validExtension(extension)
	if err != nil { return "", err }

	// check if media is banned
	mediaPath := path
	if mediaType == db.MEDIA_VIDEO {
		mediaPath = config.Cfg.Media.Tmp + "/frame_" + name + ".png"
		if err := extractFrame(path, mediaPath); err != nil {
			return "", err
		}
		defer os.Remove(mediaPath)
	}
	f, err := os.Open(mediaPath)
	if err != nil { return "", err }
	if err := isImageBanned(f); err != nil { return "", err }

	// clean up the metadata
	out := config.Cfg.Media.Tmp + "/clean_" + name + extension
	if mediaType == db.MEDIA_PICTURE && extension != ".gif" {
		if err := cleanImage(path, out); err != nil { return "", err }
		os.Remove(path)
		defer os.Remove(out)
	} else {
		out = path
	}

	// rename to the hash of itself
	hash, err := hashFile(out)
	if err != nil { return "", err }
	if config.Cfg.Media.InDatabase { // store media in database
		tn := config.Cfg.Media.Tmp + "/thumbnail_" + hash + ".png"
		src := out
		if mediaType == db.MEDIA_VIDEO {
			src = config.Cfg.Media.Tmp + "/frame_" + hash + ".png"
			if err := extractFrame(out, src); err != nil {
				return "", err
			}
			defer os.Remove(src)
		}
		if err := thumbnail(src, tn); err != nil { return "", err }
		defer os.Remove(tn)
		tn_data, err := os.ReadFile(tn)
		if err != nil { return "", err }
		data, err := os.ReadFile(out)
		if err != nil { return "", err }
		toApprove, err := db.AddMedia(data, tn_data, mediaType,
			hash, mime.String(), approved, spoiler)
		if err != nil { return "", err }
		if toApprove {
			err = notify(hash)
		}
		return hash + extension, err
	}
	toApprove, err := db.AddMedia(nil, nil, mediaType,
		hash, mime.String(), approved, spoiler)
	if toApprove && err == nil {
		err = notify(hash)
	}
	if err != nil { return "", err }
	media := config.Cfg.Media.Path + "/" + hash + extension
	err = move(out, media)
	if err != nil { return "", err }

	// create thumbnail
	if mediaType == db.MEDIA_VIDEO {
		dst := config.Cfg.Media.Tmp + "/frame_" + hash + ".png"
		if err := extractFrame(media, dst); err != nil {
			return "", err
		}
		media = dst
		defer os.Remove(media)
	}
	err = thumbnail(media,
		config.Cfg.Media.Path + "/thumbnail/" + hash + ".png");
	if err != nil { return "", err }

	return hash + extension, nil
}

func extractFrame(in string, out string) error {
	var c *exec.Cmd
	if strings.HasSuffix(in, ".gif") {
		c = exec.Command(
			"ffmpeg", "-i", in, out,
		)
	} else {
		c = exec.Command(
			"ffmpeg", "-i", in, "-vf", "select=eq(n\\,34)",
			"-vframes", "1", out,
		)
	}
	c.Stderr = os.Stderr
	c.Stdout = nil
	c.Run()
	_, err := os.Stat(out)
	return err
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

func mediaReader(hash string) (io.Reader, error) {
	isPicture := isMedia(hash, db.MEDIA_PICTURE)
	if !config.Cfg.Media.InDatabase {
		if !isPicture {
			return os.Open(config.Cfg.Media.Path +
					"/thumbnail/" + hash + ".png")
		}
		post, err := db.GetPostFromMedia(hash)
		if err != nil { return nil, err }
		return os.Open(config.Cfg.Media.Path + "/" + post.Media)
	}
	var data []byte
	var err error
	if isPicture {
		data, _, err = db.GetMediaData(hash)
	} else {
		data, err = db.GetThumbnail(hash)
	}
	if err != nil { return nil, err }
	r := bytes.NewReader(data)
	return r, nil
}
