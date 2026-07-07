package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"techguild-backend/src/config"
	"techguild-backend/src/database/postgres"
	"techguild-backend/src/dto"
	"techguild-backend/src/services"

	"github.com/gin-gonic/gin"
)

type GoogleUser struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
}
type GitHubUser struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}
func GoogleLogin(c *gin.Context) {

	url := config.GoogleOAuthConfig.AuthCodeURL("state-token")

	c.Redirect(http.StatusTemporaryRedirect, url)
}

func GoogleCallback(c *gin.Context) {

	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "authorization code not found",
		})
		return
	}

	token, err := config.GoogleOAuthConfig.Exchange(context.Background(), code)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "failed to exchange token",
		})
		return
	}

	client := config.GoogleOAuthConfig.Client(context.Background(), token)

	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "failed to fetch google user",
		})
		return
	}
	defer resp.Body.Close()

	var googleUser GoogleUser

	if err := json.NewDecoder(resp.Body).Decode(&googleUser); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to decode google user",
		})
		return
	}

	oauthService := services.NewOAuthService(postgres.RedisDB)

	result, err := oauthService.GoogleLogin(dto.GoogleLoginRequest{
		GoogleID: googleUser.ID,
		Email:    googleUser.Email,
		FullName: googleUser.Name,
		Picture:  googleUser.Picture,
	})

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}
func GitHubLogin(c *gin.Context) {

	url := config.GitHubOAuthConfig.AuthCodeURL("state-token")

	c.Redirect(http.StatusTemporaryRedirect, url)
}
func GitHubCallback(c *gin.Context) {

	code := c.Query("code")

	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "authorization code missing",
		})
		return
	}

	token, err := config.GitHubOAuthConfig.Exchange(context.Background(), code)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "failed to exchange token",
		})
		return
	}

	client := config.GitHubOAuthConfig.Client(context.Background(), token)

	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "failed to fetch github user",
		})
		return
	}
	defer resp.Body.Close()

	var githubUser GitHubUser

	if err := json.NewDecoder(resp.Body).Decode(&githubUser); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to decode github user",
		})
		return
	}

	oauthService := services.NewOAuthService(postgres.RedisDB)

	result, err := oauthService.GitHubLogin(dto.GitHubLoginRequest{
		GitHubID: strconv.FormatInt(githubUser.ID, 10),
		Email:    githubUser.Email,
		FullName: githubUser.Name,
		Avatar:   githubUser.AvatarURL,
	})

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}