package middleware

import (
	"errors"
	"log"
	"net/http"
	"strings"
	"techguild-backend/src/database/postgres"
	"techguild-backend/src/repository"
	"techguild-backend/src/utils"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret []byte

func getJwtSecret() []byte {
	return []byte("your-access-secret")
}

type Claims struct {
	UserID string `json:"user_id"`
	Type   string `json:"type"`
	jwt.RegisteredClaims
}

func AuthMiddleware() gin.HandlerFunc {
	blacklistRepo := repository.NewTokenBlacklistRepository(postgres.RedisDB)

	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header missing"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return utils.GetAccessSecret(), nil
		})

		if err != nil || !token.Valid {
			log.Println("JWT ERROR:", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		claims := token.Claims.(*Claims)
		if claims.Type != "access" {
			c.JSON(http.StatusUnauthorized, gin.H{"error:": "Invalid token type"})
			c.Abort()
			return
		}

		// reject blacklist access-token
		tokenHash := utils.HashToken(tokenString)
		blacklisted, err := blacklistRepo.IsBlacklisted(tokenHash)
		if err != nil {
			log.Println("Blacklist check error:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			c.Abort()
			return
		}
		if blacklisted {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "token has been revoked"})
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("access_token", tokenString)
		c.Next()
	}
}
