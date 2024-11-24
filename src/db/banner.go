package db

func AddBanner(data []byte) error {
	return db.Create(&Banner{Data: data}).Error
}

func RemoveBanner(id int) error {
	return db.Delete(&Banner{}, id).Error
}

func GetAllBanners() ([]uint, error) {
	var banners []Banner
	if err := db.Select("id").Find(&banners).Error; err != nil {
		return nil, err
	}
	ids := make([]uint, len(banners))
	for i, v := range banners { ids[i] = v.ID }
	return ids, nil
}

func GetBanner(id uint) (Banner, error) {
	var v Banner
	err := db.First(&v, id).Error
	return v, err
}
