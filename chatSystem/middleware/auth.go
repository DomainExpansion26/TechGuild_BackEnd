package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"strings"

	"techguild-chat/config"
	"techguild-chat/utils"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

type AuthClaims struct {
	UserID    string `json:"user_id"`
	SessionID string `json:"session_id"`
	jwt.RegisteredClaims
}

func getJwtAccessSecret() []byte {
	secret := os.Getenv("JWT_ACCESS_SECRET")
	if secret == "" {
		secret = os.Getenv("JWT_SECRET")
	}
	if secret == "" {
		config.FindAndLoadEnv()
		secret = os.Getenv("JWT_ACCESS_SECRET")
		if secret == "" {
			secret = os.Getenv("JWT_SECRET")
		}
	}
	if secret == "" {
		return []byte("temporary-default-secret-for-test-environments")
	}
	return []byte(secret)
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// AuthMiddlewareHuma validates bearer JWT token and attaches user ID to context
func AuthMiddlewareHuma(api huma.API, redisClient *redis.Client) func(ctx huma.Context, next func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		authHeader := ctx.Header("Authorization")
		if authHeader == "" {
			huma.WriteErr(api, ctx, http.StatusUnauthorized, "Authorization header missing")
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			huma.WriteErr(api, ctx, http.StatusUnauthorized, "Invalid authorization header format (Bearer required)")
			return
		}

		// Token revocation check via Redis blacklist
		if redisClient != nil {
			tokenHash := hashToken(tokenString)
			blacklisted, err := redisClient.Exists(ctx.Context(), "blacklist:"+tokenHash).Result()
			if err == nil && blacklisted > 0 {
				huma.WriteErr(api, ctx, http.StatusUnauthorized, "Token has been revoked")
				return
			}
		}

		token, err := jwt.ParseWithClaims(tokenString, &AuthClaims{}, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return getJwtAccessSecret(), nil
		})

		if err != nil || !token.Valid {
			huma.WriteErr(api, ctx, http.StatusUnauthorized, "Invalid or expired token")
			return
		}

		claims, ok := token.Claims.(*AuthClaims)
		if !ok || claims.UserID == "" {
			huma.WriteErr(api, ctx, http.StatusUnauthorized, "Invalid token claims")
			return
		}

		newCtx := huma.WithValue(ctx, utils.UserIDKey, claims.UserID)
		newCtx = huma.WithValue(newCtx, utils.SessionIDKey, claims.SessionID)
		next(newCtx)
	}
}

// AuthMiddlewareGin validates bearer token for raw Gin routes
func AuthMiddlewareGin(redisClient *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header missing"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		if redisClient != nil {
			tokenHash := hashToken(tokenString)
			blacklisted, err := redisClient.Exists(c.Request.Context(), "blacklist:"+tokenHash).Result()
			if err == nil && blacklisted > 0 {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Token has been revoked"})
				c.Abort()
				return
			}
		}

		token, err := jwt.ParseWithClaims(tokenString, &AuthClaims{}, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return getJwtAccessSecret(), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(*AuthClaims)
		if !ok || claims.UserID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("session_id", claims.SessionID)
		c.Next()
	}
}
