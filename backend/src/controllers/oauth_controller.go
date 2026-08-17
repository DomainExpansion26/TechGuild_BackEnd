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
	"techguild-backend/src/utils"
	"time"

	_ "techguild-backend/src/swagger"

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

// GitHubEmail — FIX (BUG 5): new struct to parse /user/emails response
type GitHubEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

const oauthHTTPTimeout = 10 * time.Second

// GoogleLogin godoc
// @Summary Login with Google
// @Description Redirects the user to Google's OAuth consent screen.
// @Tags OAuth
// @Produce json
// @Success 307 "Redirect to Google OAuth"
// @Router /oauth/google/login [get]
func GoogleLogin(c *gin.Context) {

	state, err := utils.GenerateOAuthState()
	if err != nil {
		c.JSON(
			http.StatusInternalServerError, gin.H{
				"error": "failed to generate state",
			})
		return
	}

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(
		"oauth_state",
		state,
		10*60,
		"/oauth",
		"",
		true,
		true,
	)
	url := config.GoogleOAuthConfig.AuthCodeURL(state)

	c.Redirect(http.StatusTemporaryRedirect, url)
}

// GoogleCallback godoc
// @Summary Google OAuth callback
// @Description Handles Google OAuth callback and authenticates the user.
// @Tags OAuth
// @Accept json
// @Produce json
// @Param code query string true "Google authorization code"
// @Success 200 {object} dto.LoginResponse
// @Failure 400 {object} swagger.ErrorResponse
// @Failure 500 {object} swagger.ErrorResponse
// @Router /oauth/google/callback [get]
func GoogleCallback(c *gin.Context) {
	code := c.Query("code")

	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "authorization code missing",
		})
		return
	}

	state := c.Query("state")
	if state == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "oauth state is missing",
		})
		return
	}

	storedState, err := c.Cookie("oauth_state")
	if err != nil || storedState != state {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "oauth state cookie missing",
		})
		return
	}

	// if state != storedState {
	// 	c.JSON(http.StatusBadRequest, gin.H{
	// 		"error": "Invalid OAuth state",
	// 	})
	// 	return
	// }
	ctx, cancel := context.WithTimeout(context.Background(), oauthHTTPTimeout)
	defer cancel()

	token, err := config.GoogleOAuthConfig.Exchange(ctx, code)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "failed to exchange token",
		})
		return
	}

	client := config.GoogleOAuthConfig.Client(ctx, token)

	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "failed to fetch google user",
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusBadGateway, gin.H{
			"error": "google user-info returned non-200",
		})
		return
	}

	var googleUser GoogleUser

	if err := json.NewDecoder(resp.Body).Decode(&googleUser); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to decode google user",
		})
		return
	}

	// FIX (VerifiedEmail check): reject login if Google says the email isn't verified.
	// Without this, an attacker-controlled unverified email could be trusted and
	// used to create/link an account (e.g. account takeover via email spoofing risk).
	if !googleUser.VerifiedEmail {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "google email is not verified",
		})
		return
	}

	oauthService := services.NewOAuthService(postgres.RedisDB)

	result, refreshToken, err := oauthService.GoogleLogin(dto.GoogleLoginRequest{
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
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(
		"refresh_token",
		refreshToken,
		int(utils.RefreshTokenTTL.Seconds()),
		"/",
		"",
		true,
		true,
	)
	c.JSON(http.StatusOK, result)
}

// GitHubLogin godoc
// @Summary Login with GitHub
// @Description Redirects the user to GitHub's OAuth authorization page.
// @Tags OAuth
// @Produce json
// @Success 307 "Redirect to GitHub OAuth"
// @Router /oauth/github/login [get]
func GitHubLogin(c *gin.Context) {

	state, err := utils.GenerateOAuthState()
	if err != nil {
		c.JSON(
			http.StatusInternalServerError,
			gin.H{
				"error": "failed to generate state",
			})
		return
	}

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(
		"oauth_state",
		state,
		10*60,
		"/oauth",
		"",
		true,
		true,
	)

	url := config.GitHubOAuthConfig.AuthCodeURL(state)

	c.Redirect(http.StatusTemporaryRedirect, url)
}

// GitHubCallback godoc
// @Summary GitHub OAuth callback
// @Description Handles GitHub OAuth callback and authenticates the user.
// @Tags OAuth
// @Accept json
// @Produce json
// @Param code query string true "GitHub authorization code"
// @Success 200 {object} dto.LoginResponse
// @Failure 400 {object} swagger.ErrorResponse
// @Failure 500 {object} swagger.ErrorResponse
// @Router /oauth/github/callback [get]
func GitHubCallback(c *gin.Context) {

	code := c.Query("code")

	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "authorization code missing",
		})
		return
	}

	state := c.Query("state")
	if state == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Authorization code missing",
		})
		return
	}

	storedState, err := c.Cookie("oauth_state")
	if err != nil || storedState != state {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "oauth state cookie missing or mismatched",
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), oauthHTTPTimeout)
	defer cancel()

	token, err := config.GitHubOAuthConfig.Exchange(ctx, code)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "failed to exchange token",
		})
		return
	}

	client := config.GitHubOAuthConfig.Client(ctx, token)

	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "failed to fetch github user",
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusBadGateway, gin.H{
			"error": "github userinfo returned non-200",
		})
		return
	}

	var githubUser GitHubUser

	if err := json.NewDecoder(resp.Body).Decode(&githubUser); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to decode github user",
		})
		return
	}

	// GitHub /user often omits email if private -> fetch /user/emails, pick primary+verified
	if githubUser.Email == "" {
		emailResp, err := client.Get("https://api.github.com/user/emails")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "failed to fetched github email",
			})
			return
		}
		defer emailResp.Body.Close()

		if emailResp.StatusCode != http.StatusOK {
			c.JSON(http.StatusBadGateway, gin.H{
				"error": "github email endpoints return non-200",
			})
			return
		}

		var emails []GitHubEmail
		if err := json.NewDecoder(emailResp.Body).Decode(&emails); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to decode github emails",
			})
			return
		}

		for _, e := range emails {
			if e.Primary && e.Verified {
				githubUser.Email = e.Email
				break
			}
		}

		if githubUser.Email == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "no verified primary email found on github account",
			})
			return
		}
	}

	oauthService := services.NewOAuthService(postgres.RedisDB)

	result, refreshToken, err := oauthService.GitHubLogin(dto.GitHubLoginRequest{
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

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(
		"refresh_token",
		refreshToken,
		int(utils.RefreshTokenTTL.Seconds()),
		"/",
		"",
		true,
		true,
	)

	c.JSON(http.StatusOK, result)
}
