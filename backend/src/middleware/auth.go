package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"techguild-backend/src/database/postgres"
	"techguild-backend/src/models"
	"techguild-backend/src/utils"

	"github.com/danielgtaylor/huma/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

type Claims struct {
	UserID    string `json:"user_id"`
	SessionID string `json:"session_id"`
	Type      string `json:"type"`
	jwt.RegisteredClaims
}

func AuthMiddlewareHuma(api huma.API) func(ctx huma.Context, next func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		authHeader := ctx.Header("Authorization")

		if authHeader == "" {
			huma.WriteErr(api, ctx, http.StatusUnauthorized, "Authorization header missing")
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// ✅ Blacklist check — logout kiye hue tokens reject karo.
		// Fail-closed (C2): Redis unavailable hone par 503 do, token ko
		// blindly accept mat karo.
		if postgres.RedisDB != nil {
			tokenHash := utils.HashToken(tokenString)
			blacklisted, err := postgres.RedisDB.Exists(
				ctx.Context(),
				"blacklist:"+tokenHash,
			).Result()
			if err != nil {
				huma.WriteErr(api, ctx, http.StatusServiceUnavailable, "Service temporarily unavailable")
				return
			}
			if blacklisted > 0 {
				huma.WriteErr(api, ctx, http.StatusUnauthorized, "Token has been revoked")
				return
			}
		}

		token, err := jwt.ParseWithClaims(
			tokenString,
			&Claims{},
			func(t *jwt.Token) (interface{}, error) {
				return utils.GetAccessSecret(), nil
			},
			jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		)

		if err != nil || !token.Valid {
			fmt.Println("JWT ERROR:", err)
			huma.WriteErr(api, ctx, http.StatusUnauthorized, "Invalid or expired token")
			return
		}

		claims := token.Claims.(*Claims)
		if claims.Type != "access" || claims.UserID == "" || claims.SessionID == "" {
			huma.WriteErr(api, ctx, http.StatusUnauthorized, "Invalid token claims")
			return
		}

		// ✅ Session/account revocation checks — access tokens are invalidated
		// when the session is revoked (refresh rotation) or when the user's
		// password is reset (user_revoked_before).
		// Fail-closed (C2): Redis errors → 503; only redis.Nil / absent key
		// means "not revoked".
		if postgres.RedisDB != nil {
			if revoked, err := postgres.RedisDB.Exists(ctx.Context(), "session_revoked:"+claims.SessionID).Result(); err != nil {
				huma.WriteErr(api, ctx, http.StatusServiceUnavailable, "Service temporarily unavailable")
				return
			} else if revoked > 0 {
				huma.WriteErr(api, ctx, http.StatusUnauthorized, "Session has been revoked")
				return
			}
			rawTS, err := postgres.RedisDB.Get(ctx.Context(), "user_revoked_before:"+claims.UserID).Result()
			if err != nil && err != redis.Nil {
				huma.WriteErr(api, ctx, http.StatusServiceUnavailable, "Service temporarily unavailable")
				return
			}
			if err == nil {
				if revokedBefore, convErr := strconv.ParseInt(rawTS, 10, 64); convErr == nil {
					if claims.IssuedAt == nil || claims.IssuedAt.Time.Unix() <= revokedBefore {
						huma.WriteErr(api, ctx, http.StatusUnauthorized, "Session has been revoked")
						return
					}
				}
			}
		}

		newCtx := huma.WithValue(ctx, utils.UserIDKey, claims.UserID)
		newCtx = huma.WithValue(newCtx, utils.SessionIDKey, claims.SessionID)
		next(newCtx)
	}
}

func AdminMiddlewareHuma(api huma.API) func(ctx huma.Context, next func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		userID, _ := ctx.Context().Value(utils.UserIDKey).(string)
		if userID == "" {
			huma.WriteErr(api, ctx, http.StatusUnauthorized, "Unauthorized: user ID missing")
			return
		}

		var user models.User
		if err := postgres.DB.Where("id = ?", userID).First(&user).Error; err != nil {
			huma.WriteErr(api, ctx, http.StatusUnauthorized, "Unauthorized: user not found")
			return
		}

		if user.AccountType == nil || *user.AccountType != models.AccountTypeAdmin {
			huma.WriteErr(api, ctx, http.StatusForbidden, "Forbidden: Admin access required")
			return
		}

		next(ctx)
	}
}

func ClientMiddlewareHuma(api huma.API) func(ctx huma.Context, next func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		userID, _ := ctx.Context().Value(utils.UserIDKey).(string)
		if userID == "" {
			huma.WriteErr(api, ctx, http.StatusUnauthorized, "Unauthorized: user ID missing")
			return
		}

		var user models.User
		if err := postgres.DB.Where("id = ?", userID).First(&user).Error; err != nil {
			huma.WriteErr(api, ctx, http.StatusUnauthorized, "Unauthorized: user not found")
			return
		}

		if user.AccountType == nil {
			huma.WriteErr(api, ctx, http.StatusForbidden, "Forbidden: Client access required")
			return
		}

		accType := string(*user.AccountType)
		if accType != string(models.AccountTypeClientAdmin) &&
			accType != string(models.AccountTypeClient) &&
			accType != string(models.AccountTypeClientOwner) &&
			accType != string(models.AccountTypeClientMember) {
			huma.WriteErr(api, ctx, http.StatusForbidden, "Forbidden: Client access required")
			return
		}

		next(ctx)
	}
}
