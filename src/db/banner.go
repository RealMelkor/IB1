package db

type Banner struct {
	Data		[]byte
}

func AddBanner(data []byte) error {
	return db.Create(&Banner{Data: data}).Error
}

func RemoveBanner(id int) error {
	return db.Delete(&Banner{}, id).Error
}

func GetAllBanners() ([]uint, error) {
	var ids []uint
	err := db.Model(&Banner{}).Pluck("id", &ids).Error
	return ids, err
}

func GetBanner(id uint) (Banner, error) {
	var v Banner
	err := db.First(&v, id).Error
	return v, err
}
