package middleware

import (
	"crypto/rsa"
	"fmt"
	"strings"

	"github.com/f0bima/go-core/response"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Auth middleware extracts and validates RS256 JWT tokens.
func Auth(pubKey *rsa.PublicKey) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c, "UNAUTHORIZED", "Authorization header is required")
			c.Abort()
			return
		}

		var tokenString string
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 && parts[0] == "Bearer" {
			tokenString = parts[1]
		} else if len(parts) == 1 {
			// Lenient mode for Swagger/Scalar UI if user forgets "Bearer "
			tokenString = parts[0]
		} else {
			response.Unauthorized(c, "UNAUTHORIZED", "Invalid Authorization header format")
			c.Abort()
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return pubKey, nil
		})

		if err != nil || !token.Valid {
			response.Unauthorized(c, "UNAUTHORIZED", "Invalid or expired token")
			c.Abort()
			return
		}

		// Extract claims
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			userID, _ := claims["user_id"].(string)
			email, _ := claims["email"].(string)

			// Inject to context
			c.Set("user_id", userID)
			c.Set("email", email)
		} else {
			response.Unauthorized(c, "UNAUTHORIZED", "Invalid token claims")
			c.Abort()
			return
		}

		c.Next()
	}
}
