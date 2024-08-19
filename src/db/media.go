package db

func AddMedia(data []byte, thumbnail []byte, hash string, mime string) error {
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
