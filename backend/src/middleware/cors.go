package middleware

import (
	"regexp"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"techguild-backend/src/config"
)

// vercelPreviewRegex strictly matches dynamic TechGuild QA and staging preview deployments on Vercel:
// e.g. https://techguild-6k0h0t3vd-techguild-staging.vercel.app
var vercelPreviewRegex = regexp.MustCompile(`^https://techguild(-[a-z0-9]+|-git-[a-z0-9_-]+)?-techguild-staging\.vercel\.app$`)

// CORS returns a Gin middleware that allows requests from the configured
// frontend and Zudoku origins (FRONTEND_URL + ZUDOKU_URL, comma-separated),
// local development origins, and production/staging Vercel origins.
func CORS(cfg *config.Config) gin.HandlerFunc {
	origins := []string{
		"http://localhost:3000",
		"http://localhost:5173",
		"https://techguild.vercel.app",
		"https://techguild-staging.vercel.app",
	}
	for _, raw := range []string{cfg.FrontendURL, cfg.ZudokuURL} {
		for _, o := range strings.Split(raw, ",") {
			if o = strings.TrimSpace(o); o != "" {
				origins = append(origins, o)
			}
		}
	}

	return cors.New(cors.Config{
		AllowOrigins: origins,
		AllowOriginFunc: func(origin string) bool {
			return vercelPreviewRegex.MatchString(origin)
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	})
}