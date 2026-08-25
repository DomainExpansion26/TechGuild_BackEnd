package middleware

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"techguild-backend/src/database/postgres"
	"techguild-backend/src/models"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gin-gonic/gin"
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

// ---------- Existing Gin middleware (Contracts, OAuth, Profile, etc. ke liye) ----------

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header missing"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			return getJwtSecret(), nil
		})

		if err != nil || !token.Valid {
			fmt.Println("JWT ERROR:", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		claims := token.Claims.(*Claims)
		c.Set("user_id", claims.UserID)

		c.Next()
	}
}

func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")
		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: user ID missing"})
			c.Abort()
			return
		}

		var user models.User
		if err := postgres.DB.Where("id = ?", userID).First(&user).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: user not found"})
			c.Abort()
			return
		}

		if user.AccountType == nil || *user.AccountType != models.AccountTypeAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: Admin access required"})
			c.Abort()
			return
		}

		c.Next()
	}
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
