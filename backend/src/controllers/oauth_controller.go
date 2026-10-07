package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"techguild-backend/src/config"
	"techguild-backend/src/database/postgres"
	"techguild-backend/src/dto"
	"techguild-backend/src/services"
	"techguild-backend/src/utils"

	"github.com/danielgtaylor/huma/v2"
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

type GitHubEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

const oauthHTTPTimeout = 10 * time.Second

type OAuthExchangeData struct {
	AccessToken         string         `json:"access_token"`
	RefreshToken        string         `json:"refresh_token"`
	Message             string         `json:"message"`
	ExpiresIn           int            `json:"expires_in"`
	RequiresTwoFactor   bool           `json:"requires_two_factor"`
	TemporaryToken      string         `json:"temporary_token"`
	User                *dto.OAuthUser `json:"user,omitempty"`
	CreatedAt           time.Time      `json:"created_at"`
	RequiresAccountType bool           `json:"requires_account_type"`
}

type OAuthController struct {
	cfg   *config.Config
	cache sync.Map
}

func NewOAuthController(cfg *config.Config) *OAuthController {
	return &OAuthController{
		cfg: cfg,
	}
}

var defaultOAuthController = NewOAuthController(nil)

func (c *OAuthController) getFrontendURL() string {
	frontendURL := ""
	if c != nil && c.cfg != nil && c.cfg.FrontendURL != "" {
		frontendURL = c.cfg.FrontendURL
	}
	if frontendURL == "" {
		frontendURL = os.Getenv("FRONTEND_URL")
	}
	if frontendURL == "" {
		frontendURL = "http://localhost:5173"
	}
	parts := strings.Split(frontendURL, ",")
	return strings.TrimRight(strings.TrimSpace(parts[0]), "/")
}

func (c *OAuthController) storeExchangeCode(ctx context.Context, code string, data *OAuthExchangeData) {
	data.CreatedAt = time.Now()
	if postgres.RedisDB != nil {
		bytes, err := json.Marshal(data)
		if err == nil {
			_ = postgres.RedisDB.Set(ctx, "oauth_exchange:"+code, string(bytes), 60*time.Second).Err()
		}
	}
	c.cache.Store(code, data)
}

func (c *OAuthController) consumeExchangeCode(ctx context.Context, code string) (*OAuthExchangeData, bool) {
	if code == "" {
		return nil, false
	}

	// 1. Check in-memory cache first (atomic read + delete)
	if val, ok := c.cache.LoadAndDelete(code); ok {
		if data, ok := val.(*OAuthExchangeData); ok {
			if postgres.RedisDB != nil {
				_ = postgres.RedisDB.Del(ctx, "oauth_exchange:"+code).Err()
			}
			if time.Since(data.CreatedAt) < 2*time.Minute {
				return data, true
			}
		}
	}

	// 2. Check Redis
	if postgres.RedisDB != nil {
		val, err := postgres.RedisDB.Get(ctx, "oauth_exchange:"+code).Result()
		if err == nil && val != "" {
			_ = postgres.RedisDB.Del(ctx, "oauth_exchange:"+code).Err()
			var data OAuthExchangeData
			if err := json.Unmarshal([]byte(val), &data); err == nil {
				return &data, true
			}
		}
	}

	return nil, false
}

// ---------- GoogleLogin (redirect) ----------

func (c *OAuthController) GoogleLoginHandler(ctx context.Context, input *dto.GoogleLoginInput) (*dto.GoogleLoginOutput, error) {
	state, err := utils.GenerateOAuthState()
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to generate state")
	}

	cookie := &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/oauth",
		MaxAge:   10 * 60,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}

	url := config.GoogleOAuthConfig.AuthCodeURL(state)

	return &dto.GoogleLoginOutput{
		Status:    http.StatusTemporaryRedirect,
		Location:  url,
		SetCookie: cookie.String(),
	}, nil
}

// Package-level fallback
func GoogleLoginHandler(ctx context.Context, input *dto.GoogleLoginInput) (*dto.GoogleLoginOutput, error) {
	return defaultOAuthController.GoogleLoginHandler(ctx, input)
}

// ---------- GoogleCallback ----------

func (c *OAuthController) GoogleCallbackHandler(ctx context.Context, input *dto.GoogleCallbackInput) (*dto.GoogleCallbackOutput, error) {
	frontendURL := c.getFrontendURL()

	// 1. Step 2: Check if this is an API code exchange from TechGuild frontend (OAuthCallback.jsx)
	if exchangeData, ok := c.consumeExchangeCode(ctx, input.Code); ok {
		output := &dto.GoogleCallbackOutput{
			Status: http.StatusOK,
			Body: dto.GoogleLoginResponse{
				Message:             exchangeData.Message,
				AccessToken:         exchangeData.AccessToken,
				ExpiresIn:           exchangeData.ExpiresIn,
				RequiresTwoFactor:   exchangeData.RequiresTwoFactor,
				TemporaryToken:      exchangeData.TemporaryToken,
				User:                exchangeData.User,
				RequiresAccountType: exchangeData.RequiresAccountType,
			},
		}
		if exchangeData.RefreshToken != "" {
			cookie := &http.Cookie{
				Name:     "refresh_token",
				Value:    exchangeData.RefreshToken,
				Path:     "/",
				MaxAge:   int(utils.RefreshTokenTTL.Seconds()),
				Secure:   true,
				HttpOnly: true,
				SameSite: http.SameSiteNoneMode,
			}
			output.SetCookie = cookie.String()
		}
		return output, nil
	}

	// Helper for safe error redirects
	errorRedirect := func(reason string) (*dto.GoogleCallbackOutput, error) {
		isBrowser := strings.Contains(input.Accept, "text/html") || !strings.Contains(input.Accept, "application/json")
		if isBrowser {
			return &dto.GoogleCallbackOutput{
				Status:   http.StatusFound,
				Location: fmt.Sprintf("%s/login?error=%s", frontendURL, url.QueryEscape(reason)),
			}, nil
		}
		return nil, huma.Error400BadRequest("authentication failed: " + reason)
	}

	// 2. Step 1: Handling Google's callback browser redirect
	if input.Code == "" {
		return errorRedirect("missing_code")
	}
	if input.State == "" {
		return errorRedirect("missing_state")
	}
	if input.OauthStateCookie == "" || input.OauthStateCookie != input.State {
		return errorRedirect("invalid_state")
	}

	reqCtx, cancel := context.WithTimeout(context.Background(), oauthHTTPTimeout)
	defer cancel()

	token, err := config.GoogleOAuthConfig.Exchange(reqCtx, input.Code)
	if err != nil {
		return errorRedirect("oauth_exchange_failed")
	}

	client := config.GoogleOAuthConfig.Client(reqCtx, token)

	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return errorRedirect("oauth_user_failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errorRedirect("oauth_user_failed")
	}

	var googleUser GoogleUser
	if err := json.NewDecoder(resp.Body).Decode(&googleUser); err != nil {
		return errorRedirect("oauth_decode_failed")
	}

	if !googleUser.VerifiedEmail {
		return errorRedirect("email_not_verified")
	}

	oauthService := services.NewOAuthService(postgres.RedisDB)

	ip := utils.GetClientIP(input.ForwardedFor)
	result, refreshToken, err := oauthService.GoogleLogin(dto.GoogleLoginRequest{
		GoogleID: googleUser.ID,
		Email:    googleUser.Email,
		FullName: googleUser.Name,
		Picture:  googleUser.Picture,
	}, input.UserAgent, ip, input.UserAgent)
	if err != nil {
		return errorRedirect("login_failed")
	}

	exchangeCode, err := utils.GenerateOAuthState()
	if err != nil {
		return errorRedirect("exchange_code_generation_failed")
	}

	// 2FA required
	if result.RequiresTwoFactor {
		c.storeExchangeCode(ctx, exchangeCode, &OAuthExchangeData{
			AccessToken:         "",
			RefreshToken:        "",
			Message:             result.Message,
			RequiresTwoFactor:   true,
			TemporaryToken:      result.TemporaryToken,
			User:                result.User,
			RequiresAccountType: result.RequiresAccountType,
		})
		redirectURL := fmt.Sprintf("%s/oauth/google/callback?code=%s&state=%s", frontendURL, exchangeCode, url.QueryEscape(input.State))
		return &dto.GoogleCallbackOutput{
			Status:   http.StatusFound,
			Location: redirectURL,
		}, nil
	}

	// Normal login: store exchange code with tokens in cache/Redis for 60s
	c.storeExchangeCode(ctx, exchangeCode, &OAuthExchangeData{
		AccessToken:         result.AccessToken,
		RefreshToken:        refreshToken,
		Message:             result.Message,
		ExpiresIn:           result.ExpiresIn,
		RequiresTwoFactor:   false,
		TemporaryToken:      "",
		User:                result.User,
		RequiresAccountType: result.RequiresAccountType,
	})

	cookie := &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     "/",
		MaxAge:   int(utils.RefreshTokenTTL.Seconds()),
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteNoneMode,
	}

	redirectURL := fmt.Sprintf("%s/oauth/google/callback?code=%s&state=%s", frontendURL, exchangeCode, url.QueryEscape(input.State))

	return &dto.GoogleCallbackOutput{
		Status:    http.StatusFound,
		Location:  redirectURL,
		SetCookie: cookie.String(),
	}, nil
}

// Package-level fallback
func GoogleCallbackHandler(ctx context.Context, input *dto.GoogleCallbackInput) (*dto.GoogleCallbackOutput, error) {
	return defaultOAuthController.GoogleCallbackHandler(ctx, input)
}

// ---------- GitHubLogin (redirect) ----------

func (c *OAuthController) GitHubLoginHandler(ctx context.Context, input *dto.GitHubLoginInput) (*dto.GitHubLoginOutput, error) {
	state, err := utils.GenerateOAuthState()
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to generate state")
	}

	cookie := &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/oauth",
		MaxAge:   10 * 60,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}

	url := config.GitHubOAuthConfig.AuthCodeURL(state)

	return &dto.GitHubLoginOutput{
		Status:    http.StatusTemporaryRedirect,
		Location:  url,
		SetCookie: cookie.String(),
	}, nil
}

// Package-level fallback
func GitHubLoginHandler(ctx context.Context, input *dto.GitHubLoginInput) (*dto.GitHubLoginOutput, error) {
	return defaultOAuthController.GitHubLoginHandler(ctx, input)
}

// ---------- GitHubCallback ----------

func (c *OAuthController) GitHubCallbackHandler(ctx context.Context, input *dto.GitHubCallbackInput) (*dto.GitHubCallbackOutput, error) {
	frontendURL := c.getFrontendURL()

	// 1. Step 2: Check if this is an API code exchange from TechGuild frontend (OAuthCallback.jsx)
	if exchangeData, ok := c.consumeExchangeCode(ctx, input.Code); ok {
		output := &dto.GitHubCallbackOutput{
			Status: http.StatusOK,
			Body: dto.GitHubLoginResponse{
				Message:             exchangeData.Message,
				AccessToken:         exchangeData.AccessToken,
				ExpiresIn:           exchangeData.ExpiresIn,
				RequiresTwoFactor:   exchangeData.RequiresTwoFactor,
				TemporaryToken:      exchangeData.TemporaryToken,
				User:                exchangeData.User,
				RequiresAccountType: exchangeData.RequiresAccountType,
			},
		}
		if exchangeData.RefreshToken != "" {
			cookie := &http.Cookie{
				Name:     "refresh_token",
				Value:    exchangeData.RefreshToken,
				Path:     "/",
				MaxAge:   int(utils.RefreshTokenTTL.Seconds()),
				Secure:   true,
				HttpOnly: true,
				SameSite: http.SameSiteNoneMode,
			}
			output.SetCookie = cookie.String()
		}
		return output, nil
	}

	errorRedirect := func(reason string) (*dto.GitHubCallbackOutput, error) {
		isBrowser := strings.Contains(input.Accept, "text/html") || !strings.Contains(input.Accept, "application/json")
		if isBrowser {
			return &dto.GitHubCallbackOutput{
				Status:   http.StatusFound,
				Location: fmt.Sprintf("%s/login?error=%s", frontendURL, url.QueryEscape(reason)),
			}, nil
		}
		return nil, huma.Error400BadRequest("authentication failed: " + reason)
	}

	if input.Code == "" {
		return errorRedirect("missing_code")
	}
	if input.State == "" {
		return errorRedirect("missing_state")
	}
	if input.OauthStateCookie == "" || input.OauthStateCookie != input.State {
		return errorRedirect("invalid_state")
	}

	reqCtx, cancel := context.WithTimeout(context.Background(), oauthHTTPTimeout)
	defer cancel()

	token, err := config.GitHubOAuthConfig.Exchange(reqCtx, input.Code)
	if err != nil {
		return errorRedirect("oauth_exchange_failed")
	}

	client := config.GitHubOAuthConfig.Client(reqCtx, token)

	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		return errorRedirect("oauth_user_failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errorRedirect("oauth_user_failed")
	}

	var githubUser GitHubUser
	if err := json.NewDecoder(resp.Body).Decode(&githubUser); err != nil {
		return errorRedirect("oauth_decode_failed")
	}

	if githubUser.Email == "" {
		emailResp, err := client.Get("https://api.github.com/user/emails")
		if err != nil {
			return errorRedirect("oauth_email_failed")
		}
		defer emailResp.Body.Close()

		if emailResp.StatusCode != http.StatusOK {
			return errorRedirect("oauth_email_failed")
		}

		var emails []GitHubEmail
		if err := json.NewDecoder(emailResp.Body).Decode(&emails); err != nil {
			return errorRedirect("oauth_decode_failed")
		}

		for _, e := range emails {
			if e.Primary && e.Verified {
				githubUser.Email = e.Email
				break
			}
		}

		if githubUser.Email == "" {
			return errorRedirect("email_not_verified")
		}
	}

	oauthService := services.NewOAuthService(postgres.RedisDB)

	ip := utils.GetClientIP(input.ForwardedFor)
	result, refreshToken, err := oauthService.GitHubLogin(dto.GitHubLoginRequest{
		GitHubID: strconv.FormatInt(githubUser.ID, 10),
		Email:    githubUser.Email,
		FullName: githubUser.Name,
		Avatar:   githubUser.AvatarURL,
	}, input.UserAgent, ip, input.UserAgent)
	if err != nil {
		return errorRedirect("login_failed")
	}

	exchangeCode, err := utils.GenerateOAuthState()
	if err != nil {
		return errorRedirect("exchange_code_generation_failed")
	}

	// 2FA required
	if result.RequiresTwoFactor {
		c.storeExchangeCode(ctx, exchangeCode, &OAuthExchangeData{
			AccessToken:         "",
			RefreshToken:        "",
			Message:             result.Message,
			RequiresTwoFactor:   true,
			TemporaryToken:      result.TemporaryToken,
			User:                result.User,
			RequiresAccountType: result.RequiresAccountType,
		})
		redirectURL := fmt.Sprintf("%s/oauth/github/callback?code=%s&state=%s", frontendURL, exchangeCode, url.QueryEscape(input.State))
		return &dto.GitHubCallbackOutput{
			Status:   http.StatusFound,
			Location: redirectURL,
		}, nil
	}

	c.storeExchangeCode(ctx, exchangeCode, &OAuthExchangeData{
		AccessToken:         result.AccessToken,
		RefreshToken:        refreshToken,
		Message:             result.Message,
		ExpiresIn:           result.ExpiresIn,
		RequiresTwoFactor:   false,
		TemporaryToken:      "",
		User:                result.User,
		RequiresAccountType: result.RequiresAccountType,
	})

	cookie := &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     "/",
		MaxAge:   int(utils.RefreshTokenTTL.Seconds()),
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteNoneMode,
	}

	redirectURL := fmt.Sprintf("%s/oauth/github/callback?code=%s&state=%s", frontendURL, exchangeCode, url.QueryEscape(input.State))

	return &dto.GitHubCallbackOutput{
		Status:    http.StatusFound,
		Location:  redirectURL,
		SetCookie: cookie.String(),
	}, nil
}

// Package-level fallback
func GitHubCallbackHandler(ctx context.Context, input *dto.GitHubCallbackInput) (*dto.GitHubCallbackOutput, error) {
	return defaultOAuthController.GitHubCallbackHandler(ctx, input)
}
