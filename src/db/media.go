package db

import (
	"time"
	"os"
	"log"
	"path/filepath"

	"IB1/config"
)

func AddMedia(data []byte, thumbnail []byte, hash string, mime string) error {
	var media Media
	var count int64
	db.First(&media, "hash = ?", hash).Count(&count)
	if count > 0 { return nil }
	return db.Create(&Media{
		Hash: hash, Mime: mime, Data: data, Thumbnail: thumbnail,
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
