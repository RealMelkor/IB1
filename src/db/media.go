package db

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/corona10/goimagehash"
	"gorm.io/gorm"

	"IB1/config"
)

type MediaType uint8

const (
	MEDIA_PICTURE MediaType = iota
	MEDIA_VIDEO
	MEDIA_AUDIO
)

type Media struct {
	Hash          string `gorm:"unique"`
	Mime          string
	Data          []byte
	Thumbnail     []byte
	Approved      bool
	HideThumbnail bool
	Type          MediaType
}

type BannedImage struct {
	gorm.Model
	Hash int64
	Kind int
}

type ApprovalBypass struct {
	gorm.Model
	CRUD[ApprovalBypass]
	Secret string
	Hash   string
}

func (ApprovalBypass) Delete(secret string) error {
	return db.Where("secret = ?", secret).Delete(&ApprovalBypass{}).Error
}

func CanBypassApproval(hash string, secret string) error {
	var count int64
	err := db.Model(&ApprovalBypass{}).
		Where("hash = ? AND secret = ?", hash, secret).
		Count(&count).Error
	if err != nil {
		return err
	}
	if count == 0 {
		return errNeedPrivilege
	}
	return nil
}

func AddMedia(data []byte, thumbnail []byte, mediaType MediaType, hash string,
	mime string, approved bool, spoiler bool) (bool, error) {
	var media Media
	var count int64
	db.First(&media, "hash = ?", hash).Count(&count)
	if count > 0 {
		return false, nil
	}
	err := db.Create(&Media{
		Hash: hash, Mime: mime, Data: data, Thumbnail: thumbnail,
		Approved: approved, Type: mediaType, HideThumbnail: spoiler,
	}).Error
	return err == nil && !approved, err
}

func GetThumbnail(hash string) ([]byte, error) {
	var media Media
	err := db.Select("thumbnail").First(&media, "hash = ?", hash).Error
	if err != nil {
		return nil, err
	}
	return media.Thumbnail, nil
}

func GetMediaData(hash string) ([]byte, string, error) {
	var media Media
	err := db.Select("data", "mime").First(&media, "hash = ?", hash).Error
	if err != nil {
		return nil, "", err
	}
	return media.Data, media.Mime, nil
}

func cleanOrphanMedias() error {
	join := "LEFT OUTER JOIN posts b ON a.hash = b.media_hash " +
		"WHERE b.media_hash IS NULL"
	query := "SELECT a.hash FROM media a " + join
	if !config.Cfg.Media.InDatabase {
		var orphans []Media
		if err := db.Raw(query).Scan(&orphans).Error; err != nil {
			return err
		}
		if len(orphans) == 0 {
			return nil
		}
		for _, v := range orphans {
			files, err := filepath.Glob(
				config.Cfg.Media.Path + "/" + v.Hash + ".*")
			if err != nil {
				continue
			}
			for _, v := range files {
				os.Remove(v)
			}
			os.Remove(config.Cfg.Media.Path +
				"/thumbnail/" + v.Hash + ".png")
		}
	}
	if dbType == TYPE_MYSQL {
		return db.Exec("DELETE a FROM media a " + join).Error
	}
	return db.Exec("DELETE FROM media WHERE hash IN (" + query + ")").Error
}

func cleanMediaTask() {
	for {
		if err := cleanOrphanMedias(); err != nil {
			log.Println(err)
		}
		time.Sleep(time.Hour)
	}
}

func GetPendingApproval() (string, string, error) {
	var media Media
	err := db.First(&media, "approved = 0").Error
	if err != nil {
		err = db.First(&media, "approved IS NULL").Error
	}
	return media.Hash, media.Mime, err
}

func Approve(hash string) error {
	return db.Model(&Media{}).Where("hash = ?", hash).
		Update("approved", true).Error
}

func ApproveAll() error {
	return db.Model(&Media{}).Where("1 = 1").Update("approved", true).Error
}

func HasUnapproved() bool {
	return db.Where("approved = ?", false).
		First(&Media{}).RowsAffected != 0
}

func RemoveMedia(hash string) error {
	if !config.Cfg.Media.InDatabase {
		files, err := filepath.Glob(
			config.Cfg.Media.Path + "/" + hash + ".*")
		if err != nil {
			return err
		}
		for _, v := range files {
			os.Remove(v)
		}
		os.Remove(
			config.Cfg.Media.Path + "/thumbnail/" + hash + ".png")
	}
	return db.Where("hash = ?", hash).Delete(&Media{}).Error
}

const NoYetApproved = "media is not yet approved"

func GetMedia(hash string) (Media, error) {
	var media Media
	err := db.First(&media, "hash = ?", hash).Error
	return media, err
}

func HasSpoiler(hash string) (bool, error) {
	var media Media
	err := db.First(&media, "hash = ?", hash).Error
	if err != nil {
		return false, err
	}
	return media.HideThumbnail, nil
}

func ToggleSpoiler(hash string) error {
	var media Media
	err := db.First(&media, "hash = ?", hash).Error
	if err != nil {
		return err
	}
	return db.Model(&Media{}).Where("hash = ?", hash).
		Update("hide_thumbnail", !media.HideThumbnail).Error
}

func IsImageBanned(hash goimagehash.ImageHash) error {
	rows, err := db.Model(&BannedImage{}).Select("hash, kind").Rows()
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var v BannedImage
		if err := db.ScanRows(rows, &v); err != nil {
			return err
		}
		img := goimagehash.NewImageHash(
			uint64(v.Hash), goimagehash.Kind(v.Kind))
		distance, err := hash.Distance(img)
		if err != nil {
			return err
		}
		if distance < config.Cfg.Media.ImageThreshold {
			return errors.New("banned image")
		}
	}
	return nil
}

func BanImage(hash goimagehash.ImageHash) error {
	return db.Create(&BannedImage{
		Hash: int64(hash.GetHash()), Kind: int(hash.GetKind()),
	}).Error
}

func GetBannedImages() ([]BannedImage, error) {
	var v []BannedImage
	err := db.Find(&v).Error
	return v, err
}

func AddBannedImage(hash int64) error {
	return db.Create(&BannedImage{Hash: hash}).Error
}

func RemoveBannedImage(hash int64) error {
	return db.Where("hash = ?", hash).Delete(&BannedImage{}).Error
}

func Extract(path string) error {
	rows, err := db.Table("media").
		Select("media.hash, media.data, media.thumbnail, posts.media").
		Joins("inner join posts on media.hash = posts.media_hash").
		Rows()
	if err != nil {
		return err
	}
	err = os.MkdirAll(path+"/thumbnail", 0755)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var v struct {
			Hash      string
			Data      []byte
			Thumbnail []byte
			Media     string
		}

		if err := db.ScanRows(rows, &v); err != nil {
			return err
		}
		if len(v.Data) == 0 {
			continue
		}

		f, err := os.Create(path + "/" + v.Media)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = f.Write(v.Data)
		if err != nil {
			return err
		}
		f.Close()

		f, err = os.Create(path + "/thumbnail/" + v.Hash + ".png")
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = f.Write(v.Thumbnail)
		if err != nil {
			return err
		}
		f.Close()
	}
	return nil
}

func Load(path string) error {
	dir, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	for _, v := range dir {
		if !v.Type().IsRegular() {
			continue
		}
		hash := strings.Split(v.Name(), ".")[0]
		data, err := os.ReadFile(path + "/" + v.Name())
		if err != nil {
			return err
		}
		thumbnail, err := os.ReadFile(
			path + "/thumbnail/" + hash + ".png")
		if err != nil {
			return err
		}
		db.Model(&Media{}).Where("hash = ?", hash).Updates(
			map[string]interface{}{
				"data": data, "thumnbail": thumbnail,
			})
	}
	return nil
}
