package middleware

import (
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"techguild-backend/src/config"
)

// CORS returns a Gin middleware that allows requests from the configured
// frontend and Zudoku origins (FRONTEND_URL + ZUDOKU_URL, comma-separated).
// Falls back to the local dev frontend when neither is set.
func CORS(cfg *config.Config) gin.HandlerFunc {
	origins := []string{"http://localhost:3000"}
	for _, raw := range []string{cfg.FrontendURL, cfg.ZudokuURL} {
		for _, o := range strings.Split(raw, ",") {
			if o = strings.TrimSpace(o); o != "" {
				origins = append(origins, o)
			}
		}
	}

	return cors.New(cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	})
}
