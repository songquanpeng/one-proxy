package controller

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"net/http"
	"one-proxy/common"
	"one-proxy/model"
	"one-proxy/subscription"
	"strconv"
	"time"
)

func GetAllProfiles(c *gin.Context) {
	p, _ := strconv.Atoi(c.Query("p"))
	if p < 0 {
		p = 0
	}
	files, err := model.GetAllProfiles(p*common.ItemsPerPage, common.ItemsPerPage)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    files,
	})
	return
}

func SearchProfiles(c *gin.Context) {
	keyword := c.Query("keyword")
	files, err := model.SearchProfiles(keyword)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    files,
	})
	return
}

func GetProfile(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	profile, err := model.GetProfileById(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    profile,
	})
	return
}

func GetProfileByToken(c *gin.Context) {
	token := c.Param("token")
	profile, err := model.GetProfileByToken(token)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	if profile.Status != model.ProfileStatusEnabled {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "Profile is not enabled",
		})
		return
	}
	var cache *model.ProfileCache
	if profile.FetchMode == model.ProfileFetchModeProxy {
		cache, err = subscription.DefaultService.FetchDirect(c.Request.Context(), profile, c.Request.Header)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"success": false, "message": err.Error()})
			return
		}
	} else {
		cache, err = subscription.DefaultService.FetchAndCache(c.Request.Context(), profile, c.Request.Header)
		if err != nil {
			fetchErr := err
			cache, err = subscription.DefaultService.Cached(profile.Id, c.GetHeader("User-Agent"))
			if err == nil {
				c.Header("X-One-Proxy-Stale", "true")
				c.Header("Warning", `110 one-proxy "Response is stale"`)
			} else {
				err = fetchErr
			}
		}
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{
				"success": false,
				"message": "订阅缓存尚未生成，且源站拉取失败: " + err.Error(),
			})
			return
		}
	}
	c.Header("X-One-Proxy-Mode", profile.FetchMode)
	writeCachedSubscription(c, cache)
}

func writeCachedSubscription(c *gin.Context, cache *model.ProfileCache) {
	if cache.ResponseHeaders != "" {
		var headers http.Header
		if json.Unmarshal([]byte(cache.ResponseHeaders), &headers) == nil {
			for name, values := range headers {
				for _, value := range values {
					c.Writer.Header().Add(name, value)
				}
			}
		}
	}
	if cache.ContentDisposition != "" {
		c.Header("Content-Disposition", cache.ContentDisposition)
	}
	if cache.SubscriptionUserinfo != "" {
		c.Header("Subscription-Userinfo", cache.SubscriptionUserinfo)
	}
	if cache.ProfileUpdateInterval != "" {
		c.Header("Profile-Update-Interval", cache.ProfileUpdateInterval)
	}
	if cache.ProfileWebPageURL != "" {
		c.Header("Profile-Web-Page-Url", cache.ProfileWebPageURL)
	}
	if cache.SupportURL != "" {
		c.Header("Support-Url", cache.SupportURL)
	}
	c.Header("ETag", cache.ETag)
	c.Header("Last-Modified", time.Unix(cache.FetchedTime, 0).UTC().Format(http.TimeFormat))
	c.Header("X-One-Proxy-Cache", cache.Variant)
	if c.GetHeader("If-None-Match") == cache.ETag {
		c.Status(http.StatusNotModified)
		return
	}
	c.Data(http.StatusOK, cache.ContentType, cache.Content)
}

func CreateProfile(c *gin.Context) {
	profile := model.Profile{}
	err := c.BindJSON(&profile)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	profile.CreatedTime = common.GetTimestamp()
	profile.Token = common.GetUUID()
	if err = subscription.ValidateProfile(&profile); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	err = profile.Insert()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
	return
}

func UpdateProfile(c *gin.Context) {
	profile := model.Profile{}
	err := c.BindJSON(&profile)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	if c.Query("status_only") == "true" {
		if profile.Status != model.ProfileStatusEnabled && profile.Status != model.ProfileStatusDisabled {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "未知的订阅状态"})
			return
		}
		err = model.UpdateProfileStatus(profile.Id, profile.Status)
	} else {
		if err = subscription.ValidateProfile(&profile); err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
			return
		}
		err = profile.UpdateEditableFields()
	}
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
	return
}

func DeleteProfile(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	profile := model.Profile{Id: id}
	err := profile.Delete()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
	return
}

func ResetProfile(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	profile, err := model.GetProfileById(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	profile.Token = common.GetUUID()
	err = model.UpdateProfileToken(profile.Id, profile.Token)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    profile.Token,
	})
	return
}
