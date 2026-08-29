package model

import (
	"errors"
	"gorm.io/gorm"
)

type Profile struct {
	Id             int    `json:"id"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	Status         int    `json:"status" gorm:"default:1"`
	Token          string `json:"token" gorm:"type:char(32);uniqueIndex"`
	CreatedTime    int64  `json:"created_time" gorm:"bigint"`
	URL            string `json:"url"`
	FetchMode      string `json:"fetch_mode" gorm:"type:varchar(16);default:cache"`
	LastFetchTime  int64  `json:"last_fetch_time" gorm:"bigint"`
	LastFetchError string `json:"last_fetch_error" gorm:"type:text"`
}

const (
	ProfileStatusEnabled  = 1
	ProfileStatusDisabled = 2
	ProfileFetchModeCache = "cache"
	ProfileFetchModeProxy = "proxy"
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
	return DB.Create(profile).Error
}

func (profile *Profile) UpdateEditableFields() error {
	return DB.Model(&Profile{}).Where("id = ?", profile.Id).Updates(map[string]interface{}{
		"name":        profile.Name,
		"description": profile.Description,
		"url":         profile.URL,
		"fetch_mode":  profile.FetchMode,
	}).Error
}

func UpdateProfileStatus(id int, status int) error {
	return DB.Model(&Profile{}).Where("id = ?", id).Update("status", status).Error
}

func UpdateProfileToken(id int, token string) error {
	return DB.Model(&Profile{}).Where("id = ?", id).Update("token", token).Error
}

func UpdateProfileFetchResult(id int, fetchedAt int64, fetchError string) error {
	return DB.Model(&Profile{}).Where("id = ?", id).Updates(map[string]interface{}{
		"last_fetch_time":  fetchedAt,
		"last_fetch_error": fetchError,
	}).Error
}

func UpdateProfileFetchError(id int, fetchError string) error {
	return DB.Model(&Profile{}).Where("id = ?", id).Update("last_fetch_error", fetchError).Error
}

func (profile *Profile) Delete() error {
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("profile_id = ?", profile.Id).Delete(&ProfileCache{}).Error; err != nil {
			return err
		}
		return tx.Delete(profile).Error
	})
}
