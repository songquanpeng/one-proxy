package controller

import (
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"one-proxy/common"
	"one-proxy/model"
	"strconv"
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
	req, err := http.NewRequest(c.Request.Method, profile.URL, c.Request.Body)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	req.Header = c.Request.Header.Clone()
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	for k, v := range resp.Header {
		c.Header(k, v[0])
	}
	//if resp.Header.Get("Content-Disposition") != "" {
	//	c.Header("Content-Disposition", "attachment; filename="+profile.Name)
	//}
	c.Status(resp.StatusCode)
	_, err = io.Copy(c.Writer, resp.Body)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
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
	err = profile.Update()
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
	err = profile.Update()
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
