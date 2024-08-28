package db

import (
	"errors"
)

const (
	RANK_TRUSTED = iota + 1
	RANK_MODERATOR
	RANK_ADMIN
)

func StringToRank(rank string) (int, error) {
	switch rank {
	case "trusted":
		return RANK_TRUSTED, nil
	case "moderator":
		return RANK_MODERATOR, nil
	case "admin":
		return RANK_ADMIN, nil
	}
	return -1, errors.New("invalid rank")
}

func createSession(account Account) (string, error) {
	token, err := newToken()
	if err != nil { return "", err }
	err = db.Create(&Session{
		Account: account,
		AccountID: account.ID,
		Token: token,
	}).Error
	return token, err
}

var sessions = map[string]Account{}

func GetAccountFromToken(token string) (Account, error) {
	account, ok := sessions[token]
	if ok { return account, nil }
	var session Session
	err := db.Model(session).Preload("Account").
		First(&session, "token = ?", token).Error
	if err == nil {
		session.Account.Logged = true
		sessions[token] = session.Account
	}
	return session.Account, err
}

func Disconnect(token string) error {
	err := db.Where("token = ?", token).Delete(&Session{}).Error
	if err == nil { delete(sessions, token) }
	return err
}

func CreateAccount(name string, password string, rank int) error {
	hash, err := hashPassword(password)
	if err != nil { return err }
	err = db.Create(&Account{
		Name: name,
		Password: hash,
		Rank: rank,
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
	if err := db.Find(&accounts).Error; err != nil { return nil, err }
	return accounts, nil
}

func UpdateAccount(id int, name string, password string, rank int) error {
	acc := db.Model(&Account{}).Where("id = ?", id)
	if acc.Error != nil { return acc.Error }
	if password != "" {
		var err error
		password, err = hashPassword(password)
		if err != nil { return err }
	}
	sessions = map[string]Account{}
	return acc.Updates(Account{
		Name: name, Rank: rank, Password: password}).Error
}

func RemoveAccount(id uint) error {
	err := db.Unscoped().Delete(&Account{}, id).Error
	if err != nil { return err }
	db.Where("account_id = ?", id).Delete(&Session{})
	sessions = map[string]Account{}
	return nil
}

func (account Account) HasRank(rank string) bool {
	i, err := StringToRank(rank)
	if err != nil { return false }
	return account.Rank >= i
}
