package model

import (
	"errors"
)

type Profile struct {
	Id          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      int    `json:"status" gorm:"default:1"`
	Token       string `json:"token" gorm:"type:char(32);uniqueIndex"`
	CreatedTime int64  `json:"created_time" gorm:"bigint"`
	URL         string `json:"url"`
}

const (
	ProfileStatusEnabled  = 1
	ProfileStatusDisabled = 2
)

func GetAllProfiles(startIdx int, num int) ([]*Profile, error) {
	var profiles []*Profile
	var err error
	err = DB.Order("id desc").Limit(num).Offset(startIdx).Find(&profiles).Error
	return profiles, err
}

func SearchProfiles(keyword string) (files []*Profile, err error) {
	err = DB.Find(&files).Error
	return files, err
}

func GetProfileById(id int) (*Profile, error) {
	if id == 0 {
		return nil, errors.New("id 为空！")
	}
	profile := Profile{Id: id}
	var err error = nil
	err = DB.First(&profile, "id = ?", id).Error
	return &profile, err
}

func GetProfileByToken(token string) (*Profile, error) {
	if token == "" {
		return nil, errors.New("token 为空！")
	}
	profile := Profile{Token: token}
	var err error = nil
	err = DB.First(&profile, "token = ?", token).Error
	return &profile, err
}

func (profile *Profile) Insert() error {
	var err error
	err = DB.Create(profile).Error
	return err
}

func (profile *Profile) Update() error {
	var err error
	err = DB.Model(profile).Updates(profile).Error
	return err
}

// Delete Make sure link is valid! Because we will use os.Remove to delete it!
func (profile *Profile) Delete() error {
	var err error
	err = DB.Delete(profile).Error
	return err
}
