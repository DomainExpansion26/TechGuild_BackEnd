package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
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

// ---------- GoogleLogin (redirect) ----------

func GoogleLoginHandler(ctx context.Context, input *dto.GoogleLoginInput) (*dto.GoogleLoginOutput, error) {
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

// ---------- GoogleCallback ----------

func GoogleCallbackHandler(ctx context.Context, input *dto.GoogleCallbackInput) (*dto.GoogleCallbackOutput, error) {
	if input.Code == "" {
		return nil, huma.Error400BadRequest("authorization code missing")
	}
	if input.State == "" {
		return nil, huma.Error400BadRequest("oauth state is missing")
	}
	if input.OauthStateCookie == "" || input.OauthStateCookie != input.State {
		return nil, huma.Error400BadRequest("oauth state cookie missing")
	}

	reqCtx, cancel := context.WithTimeout(context.Background(), oauthHTTPTimeout)
	defer cancel()

	token, err := config.GoogleOAuthConfig.Exchange(reqCtx, input.Code)
	if err != nil {
		return nil, huma.Error400BadRequest("failed to exchange token")
	}

	client := config.GoogleOAuthConfig.Client(reqCtx, token)

	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, huma.Error400BadRequest("failed to fetch google user")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, huma.Error502BadGateway("google user-info returned non-200")
	}

	var googleUser GoogleUser
	if err := json.NewDecoder(resp.Body).Decode(&googleUser); err != nil {
		return nil, huma.Error500InternalServerError("failed to decode google user")
	}

	if !googleUser.VerifiedEmail {
		return nil, huma.Error400BadRequest("google email is not verified")
	}

	oauthService := services.NewOAuthService(postgres.RedisDB)

	result, refreshToken, err := oauthService.GoogleLogin(dto.GoogleLoginRequest{
		GoogleID: googleUser.ID,
		Email:    googleUser.Email,
		FullName: googleUser.Name,
		Picture:  googleUser.Picture,
	})
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	cookie := &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     "/",
		MaxAge:   int(utils.RefreshTokenTTL.Seconds()),
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}

	return &dto.GoogleCallbackOutput{
		SetCookie: cookie.String(),
		Body:      *result,
	}, nil
}

// ---------- GitHubLogin (redirect) ----------

func GitHubLoginHandler(ctx context.Context, input *dto.GitHubLoginInput) (*dto.GitHubLoginOutput, error) {
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

// ---------- GitHubCallback ----------

func GitHubCallbackHandler(ctx context.Context, input *dto.GitHubCallbackInput) (*dto.GitHubCallbackOutput, error) {
	if input.Code == "" {
		return nil, huma.Error400BadRequest("authorization code missing")
	}
	if input.State == "" {
		return nil, huma.Error400BadRequest("Authorization code missing")
	}
	if input.OauthStateCookie == "" || input.OauthStateCookie != input.State {
		return nil, huma.Error400BadRequest("oauth state cookie missing or mismatched")
	}

	reqCtx, cancel := context.WithTimeout(context.Background(), oauthHTTPTimeout)
	defer cancel()

	token, err := config.GitHubOAuthConfig.Exchange(reqCtx, input.Code)
	if err != nil {
		return nil, huma.Error400BadRequest("failed to exchange token")
	}

	client := config.GitHubOAuthConfig.Client(reqCtx, token)

	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		return nil, huma.Error400BadRequest("failed to fetch github user")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, huma.Error502BadGateway("github userinfo returned non-200")
	}

	var githubUser GitHubUser
	if err := json.NewDecoder(resp.Body).Decode(&githubUser); err != nil {
		return nil, huma.Error500InternalServerError("failed to decode github user")
	}

	if githubUser.Email == "" {
		emailResp, err := client.Get("https://api.github.com/user/emails")
		if err != nil {
			return nil, huma.Error400BadRequest("failed to fetched github email")
		}
		defer emailResp.Body.Close()

		if emailResp.StatusCode != http.StatusOK {
			return nil, huma.Error502BadGateway("github email endpoints return non-200")
		}

		var emails []GitHubEmail
		if err := json.NewDecoder(emailResp.Body).Decode(&emails); err != nil {
			return nil, huma.Error500InternalServerError("failed to decode github emails")
		}

		for _, e := range emails {
			if e.Primary && e.Verified {
				githubUser.Email = e.Email
				break
			}
		}

		if githubUser.Email == "" {
			return nil, huma.Error400BadRequest("no verified primary email found on github account")
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
		return nil, huma.Error400BadRequest(err.Error())
	}

	cookie := &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     "/",
		MaxAge:   int(utils.RefreshTokenTTL.Seconds()),
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}

	return &dto.GitHubCallbackOutput{
		SetCookie: cookie.String(),
		Body:      *result,
	}, nil
}
