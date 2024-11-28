package db

import (
	"time"
	"os"
	"log"
	"path/filepath"
	"errors"

	"github.com/corona10/goimagehash"

	"IB1/config"
)

func AddMedia(data []byte, thumbnail []byte, mediaType MediaType,
		hash string, mime string, approved bool) error {
	var media Media
	var count int64
	db.First(&media, "hash = ?", hash).Count(&count)
	if count > 0 { return nil }
	return db.Create(&Media{
		Hash: hash, Mime: mime, Data: data, Thumbnail: thumbnail,
		Approved: approved, Type: mediaType,
	}).Error
}

func GetThumbnail(hash string) ([]byte, error) {
	var media Media
	err := db.Select("thumbnail").First(&media, "hash = ?", hash).Error
	if err != nil { return nil, err }
	return media.Thumbnail, nil
}

func GetMedia(hash string) ([]byte, string, error) {
	var media Media
	err := db.Select("data", "mime").First(&media, "hash = ?", hash).Error
	if err != nil { return nil, "", err }
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
		if len(orphans) == 0 { return nil }
		for _, v := range orphans {
			files, err := filepath.Glob(
				config.Cfg.Media.Path + "/" + v.Hash + ".*")
			if err != nil { continue }
			for _, v := range files { os.Remove(v) }
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
	if err != nil { err = db.First(&media, "approved IS NULL").Error }
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
		if err != nil { return err }
		for _, v := range files { os.Remove(v) }
		os.Remove(
			config.Cfg.Media.Path + "/thumbnail/" + hash + ".png")
	}
	return db.Where("hash = ?", hash).Delete(&Media{}).Error
}

const NoYetApproved = "media is not yet approved"

func IsApproved(hash string) error {
	var media Media
	err := db.First(&media, "hash = ?", hash).Error
	if err != nil { return err }
	if media.Approved { return nil }
	return errors.New(NoYetApproved)
}

func IsImageBanned(hash goimagehash.ImageHash) error {
	rows, err := db.Model(&BannedImage{}).Select("hash, kind").Rows()
	if err != nil { return err }
	defer rows.Close()
	for rows.Next() {
		var v BannedImage
		if err := db.ScanRows(rows, &v); err != nil { return err }
		img := goimagehash.NewImageHash(
				uint64(v.Hash), goimagehash.Kind(v.Kind))
		distance, err := hash.Distance(img)
		if err != nil { return err }
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
