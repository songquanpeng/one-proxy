package model

import "gorm.io/gorm/clause"

type ProfileCache struct {
	Id                    int    `json:"id"`
	ProfileId             int    `json:"profile_id" gorm:"uniqueIndex:idx_profile_variant"`
	Variant               string `json:"variant" gorm:"type:varchar(32);uniqueIndex:idx_profile_variant"`
	Content               []byte `json:"-" gorm:"type:longblob"`
	ContentType           string `json:"content_type"`
	ContentDisposition    string `json:"content_disposition"`
	SubscriptionUserinfo  string `json:"subscription_userinfo"`
	ProfileUpdateInterval string `json:"profile_update_interval"`
	ProfileWebPageURL     string `json:"profile_web_page_url"`
	SupportURL            string `json:"support_url"`
	ETag                  string `json:"etag"`
	FetchedTime           int64  `json:"fetched_time" gorm:"bigint"`
}

func UpsertProfileCache(cache *ProfileCache) error {
	return DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "profile_id"}, {Name: "variant"}},
		DoUpdates: clause.AssignmentColumns([]string{"content", "content_type", "content_disposition", "subscription_userinfo", "profile_update_interval", "profile_web_page_url", "support_url", "e_tag", "fetched_time"}),
	}).Create(cache).Error
}

func GetProfileCache(profileId int, variants ...string) (*ProfileCache, error) {
	var cache ProfileCache
	query := DB.Where("profile_id = ?", profileId)
	if len(variants) > 0 {
		query = query.Where("variant IN ?", variants)
	}
	err := query.Order("fetched_time desc").First(&cache).Error
	return &cache, err
}

func GetProfileCaches(profileId int) ([]*ProfileCache, error) {
	var caches []*ProfileCache
	err := DB.Where("profile_id = ?", profileId).Order("variant asc").Find(&caches).Error
	return caches, err
}
