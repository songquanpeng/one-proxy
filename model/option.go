package model

import (
	"gin-template/common"
	"strconv"
	"strings"
)

type Option struct {
	Key   string `json:"key" gorm:"primaryKey"`
	Value string `json:"value"`
}

func AllOption() ([]*Option, error) {
	var options []*Option
	var err error
	err = DB.Find(&options).Error
	return options, err
}

func InitOptionMap() {
	common.OptionMapRWMutex.Lock()
	common.OptionMap = make(map[string]string)
	common.OptionMap["FileUploadPermission"] = strconv.Itoa(common.FileUploadPermission)
	common.OptionMap["FileDownloadPermission"] = strconv.Itoa(common.FileDownloadPermission)
	common.OptionMap["ImageUploadPermission"] = strconv.Itoa(common.ImageUploadPermission)
	common.OptionMap["ImageDownloadPermission"] = strconv.Itoa(common.ImageDownloadPermission)
	common.OptionMap["PasswordLoginEnabled"] = strconv.FormatBool(common.PasswordLoginEnabled)
	common.OptionMap["PasswordRegisterEnabled"] = strconv.FormatBool(common.PasswordRegisterEnabled)
	common.OptionMap["EmailVerificationEnabled"] = strconv.FormatBool(common.EmailVerificationEnabled)
	common.OptionMap["GitHubOAuthEnabled"] = strconv.FormatBool(common.GitHubOAuthEnabled)
	common.OptionMap["WeChatAuthEnabled"] = strconv.FormatBool(common.WeChatAuthEnabled)
	common.OptionMap["TurnstileCheckEnabled"] = strconv.FormatBool(common.TurnstileCheckEnabled)
	common.OptionMap["RegisterEnabled"] = strconv.FormatBool(common.RegisterEnabled)
	common.OptionMap["SMTPServer"] = ""
	common.OptionMap["SMTPPort"] = strconv.Itoa(common.SMTPPort)
	common.OptionMap["SMTPAccount"] = ""
	common.OptionMap["SMTPToken"] = ""
	common.OptionMap["Notice"] = ""
	common.OptionMap["About"] = ""
	common.OptionMap["Footer"] = common.Footer
	common.OptionMap["HomePageLink"] = common.HomePageLink
	common.OptionMap["SystemName"] = common.SystemName
	common.OptionMap["ServerAddress"] = ""
	common.OptionMap["GitHubClientId"] = ""
	common.OptionMap["GitHubClientSecret"] = ""
	common.OptionMap["WeChatServerAddress"] = ""
	common.OptionMap["WeChatServerToken"] = ""
	common.OptionMap["WeChatAccountQRCodeImageURL"] = ""
	common.OptionMap["TurnstileSiteKey"] = ""
	common.OptionMap["TurnstileSecretKey"] = ""
	common.OptionMapRWMutex.Unlock()
	options, _ := AllOption()
	for _, option := range options {
		updateOptionMap(option.Key, option.Value)
	}
}

func UpdateOption(key string, value string) error {
	// Save to database first
	option := Option{
		Key: key,
	}
	// https://gorm.io/docs/update.html#Save-All-Fields
	DB.FirstOrCreate(&option, Option{Key: key})
	option.Value = value
	// Save is a combination function.
	// If save value does not contain primary key, it will execute Create,
	// otherwise it will execute Update (with all fields).
	DB.Save(&option)
	// Update OptionMap
	updateOptionMap(key, value)
	return nil
}

func updateOptionMap(key string, value string) {
	common.OptionMapRWMutex.Lock()
	defer common.OptionMapRWMutex.Unlock()
	common.OptionMap[key] = value
	if strings.HasSuffix(key, "Permission") {
		intValue, _ := strconv.Atoi(value)
		switch key {
		case "FileUploadPermission":
			common.FileUploadPermission = intValue
		case "FileDownloadPermission":
			common.FileDownloadPermission = intValue
		case "ImageUploadPermission":
			common.ImageUploadPermission = intValue
		case "ImageDownloadPermission":
			common.ImageDownloadPermission = intValue
		}
	}
	if strings.HasSuffix(key, "Enabled") {
		boolValue := value == "true"
		switch key {
		case "PasswordRegisterEnabled":
			common.PasswordRegisterEnabled = boolValue
		case "PasswordLoginEnabled":
			common.PasswordLoginEnabled = boolValue
		case "EmailVerificationEnabled":
			common.EmailVerificationEnabled = boolValue
		case "GitHubOAuthEnabled":
			common.GitHubOAuthEnabled = boolValue
		case "WeChatAuthEnabled":
			common.WeChatAuthEnabled = boolValue
		case "TurnstileCheckEnabled":
			common.TurnstileCheckEnabled = boolValue
		case "RegisterEnabled":
			common.RegisterEnabled = boolValue
		}
	}
	switch key {
	case "SMTPServer":
		common.SMTPServer = value
	case "SMTPPort":
		intValue, _ := strconv.Atoi(value)
		common.SMTPPort = intValue
	case "SMTPAccount":
		common.SMTPAccount = value
	case "SMTPToken":
		common.SMTPToken = value
	case "ServerAddress":
		common.ServerAddress = value
	case "GitHubClientId":
		common.GitHubClientId = value
	case "GitHubClientSecret":
		common.GitHubClientSecret = value
	case "Footer":
		common.Footer = value
	case "HomePageLink":
		common.HomePageLink = value
	case "SystemName":
		common.SystemName = value
	case "WeChatServerAddress":
		common.WeChatServerAddress = value
	case "WeChatServerToken":
		common.WeChatServerToken = value
	case "WeChatAccountQRCodeImageURL":
		common.WeChatAccountQRCodeImageURL = value
	case "TurnstileSiteKey":
		common.TurnstileSiteKey = value
	case "TurnstileSecretKey":
		common.TurnstileSecretKey = value
	}
}
