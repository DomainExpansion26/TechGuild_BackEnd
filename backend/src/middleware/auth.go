package middleware

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"techguild-backend/src/database/postgres"
	"techguild-backend/src/models"

	"github.com/danielgtaylor/huma/v2"
	"github.com/golang-jwt/jwt/v5"
)

func getJwtSecret() []byte {
	secret := os.Getenv("JWT_ACCESS_SECRET")
	if secret == "" {
		fmt.Println("JWT ERROR: JWT_ACCESS_SECRET is not set")
		return nil
	}
	return []byte(secret)
}

type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

// ---------- Huma middleware (Auth ke migrated routes ke liye) ----------

type ctxKey string

const UserIDKey ctxKey = "user_id"

func AuthMiddlewareHuma(api huma.API) func(ctx huma.Context, next func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		authHeader := ctx.Header("Authorization")

		if authHeader == "" {
			huma.WriteErr(api, ctx, http.StatusUnauthorized, "Authorization header missing")
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
			return getJwtSecret(), nil
		})

		if err != nil || !token.Valid {
			fmt.Println("JWT ERROR:", err)
			huma.WriteErr(api, ctx, http.StatusUnauthorized, "Invalid or expired token")
			return
		}

		claims := token.Claims.(*Claims)
		newCtx := huma.WithValue(ctx, UserIDKey, claims.UserID)
		next(newCtx)
	}
}

func AdminMiddlewareHuma(api huma.API) func(ctx huma.Context, next func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		userID, _ := ctx.Context().Value(UserIDKey).(string)
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
