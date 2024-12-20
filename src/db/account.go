package db

import (
	"errors"
	"IB1/util"
)

func GetRank(name string) (Rank, error) {
	var rank Rank
	err := db.First(&rank, "name = ?", name).Error
	return rank, err
}

func createSession(account Account) (string, error) {
	token, err := util.NewToken()
	if err != nil { return "", err }
	err = db.Create(&Session{
		Account: account,
		AccountID: account.ID,
		Token: token,
	}).Error
	return token, err
}

// cached session tokens
var sessions = util.SafeMap[Account]{}

func GetAccountFromToken(token string) (Account, error) {
	account, ok := sessions.Get(token)
	if ok { return account, nil }
	var session Session
	err := db.Model(session).Preload("Account").
		First(&session, "token = ?", token).Error
	if err == nil {
		db.First(&session.Account.Rank, session.Account.RankID)
		session.Account.Logged = true
		sessions.Set(token, session.Account)
	}
	return session.Account, err
}

func Disconnect(token string) error {
	err := db.Where("token = ?", token).Delete(&Session{}).Error
	if err == nil { sessions.Delete(token) }
	return err
}

func CreateAccount(name string, password string,
			rank string, admin bool) error {
	hash, err := hashPassword(password)
	if err != nil { return err }
	v := Rank{}
	if rank != "" {
		rank, err := GetRank(rank)
		if err != nil { return err }
		v = rank
	}
	superuser := &admin
	if admin == false { superuser = nil }
	err = db.Create(&Account{
		Name: name,
		Password: hash,
		Rank: v,
		Superuser: superuser,
	}).Error
	return err
}

func Login(name string, password string) (string, error) {
	var account Account
	err := db.First(&account, "name = ?", name).Error
	if err != nil { return "", err }
	err = comparePassword(password, account.Password)
	if err != nil { return "", err }
	token, err := createSession(account)
	return token, err
}

func AccountsCount() (int, error) {
	result := db.Find(&Account{})
	if result.Error != nil { return -1, result.Error }
	return int(result.RowsAffected), nil
}

func HasSuperuser() (bool, error) {
	result := db.Find(&Account{}, "superuser = ?", true)
	if result.Error != nil { return false, result.Error }
	return result.RowsAffected > 0, nil
}

func ChangePassword(name string, password string) error {
	var account Account
	err := db.First(&account, "name = ?", name).Error
	if err != nil { return err }
	hash, err := hashPassword(password)
	if err != nil { return err }
	return db.Model(&account).Update("password", hash).Error
}

func GetAccounts() ([]Account, error) {
	var accounts []Account
	if err := db.Preload("Rank").Find(&accounts).Error; err != nil {
		return nil, err
	}
	return accounts, nil
}

func UpdateAccount(id int, name string, password string, rank string) error {
	acc := db.Model(&Account{}).Where("id = ?", id)
	if acc.Error != nil { return acc.Error }
	v, err := GetRank(rank)
	if err != nil { return err }
	if password != "" {
		var err error
		password, err = hashPassword(password)
		if err != nil { return err }
	}
	sessions.Clear()
	return acc.Updates(Account{
		Name: name, Rank: v, Password: password}).Error
}

func RemoveAccount(id uint) error {
	err := db.Unscoped().Delete(&Account{}, id).Error
	if err != nil { return err }
	db.Where("account_id = ?", id).Delete(&Session{})
	sessions.Clear()
	return nil
}

func (account Account) IsSuperuser() bool {
	if account.Superuser == nil { return false }
	return *account.Superuser
}

func (account Account) HasPrivilege(privilege string) error {
	priv := GetPrivilege(privilege)
	if priv == NONE { return errors.New("invalid privilege") }
	return account.Can(priv)
}

func (account Account) Can(privilege Privilege) error {
	if account.IsSuperuser() { return nil }
	for _, v := range account.Rank.Privileges {
		if v == privilege { return nil }
	}
	return errors.New("privilege insufficient")
}

func (account *Account) SetTheme(name string) error {
	return db.Model(account).Updates(Account{Theme: name}).Error
}

func GetUserTheme(name string) (string, error) {
	var account Account
	err := db.First(&account, "name = ?", name).Error
	if err != nil { return "", err }
	return account.Theme, nil
}
