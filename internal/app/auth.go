package app

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Auth middleware supporting static tokens or JWT
func AuthMiddlewareFromEnv() gin.HandlerFunc {
	staticTokens := strings.Split(strings.TrimSpace(os.Getenv("STATIC_TOKENS")), ",")
	jwtSecret := strings.TrimSpace(os.Getenv("JWT_HMAC_SECRET"))

	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization"})
			return
		}
		parts := strings.Fields(auth)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format"})
			return
		}
		tokenStr := parts[1]

		// JWT path
		if jwtSecret != "" {
			_, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrTokenMalformed
				}
				return []byte(jwtSecret), nil
			}, jwt.WithLeeway(5*time.Second))
			if err == nil {
				c.Next()
				return
			}
		}

		// static tokens
		for _, t := range staticTokens {
			if tokenStr == strings.TrimSpace(t) {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
	}
}
