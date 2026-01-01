package middlewares

import (
	"net/http"
	"strings"

	"github.com/RehanAthallahAzhar/tokohobby-orders/internal/models"
	"github.com/RehanAthallahAzhar/tokohobby-orders/internal/pkg/grpc/account"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"

	"github.com/labstack/echo/v4"
)

func RequireRoles(allowedRoles ...string) echo.MiddlewareFunc {
	roleSet := make(map[string]struct{})
	for _, r := range allowedRoles {
		roleSet[r] = struct{}{}
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			role, ok := c.Get("role").(string)
			if !ok {
				return c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "Unauthorized"})
			}

			if _, allowed := roleSet[role]; !allowed {
				return c.JSON(http.StatusForbidden, models.ErrorResponse{Error: "Access denied"})
			}

			return next(c)
		}
	}
}

func AuthMiddleware(authClient *account.AuthClient, jwtSecret string, audience string, log *logrus.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return c.JSON(http.StatusUnauthorized, echo.Map{"message": "Authorization token not found"})
			}

			tokenString := authHeader
			if len(authHeader) > 7 && strings.HasPrefix(authHeader, "Bearer ") {
				tokenString = authHeader[7:]
			} else {
				return c.JSON(http.StatusBadRequest, echo.Map{"message": "Invalid token format (expected Bearer token)"})
			}

			// check token via rpc
			isValid, userID, username, role, errMsg, err := authClient.ValidateToken(tokenString)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, echo.Map{"message": "Server error during token validation"})
			}

			if !isValid {
				return c.JSON(http.StatusUnauthorized, echo.Map{"message": errMsg})
			}

			// check audience
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				return []byte(jwtSecret), nil
			}, jwt.WithAudience(audience))

			if err != nil || !token.Valid {
				log.Warnf("Invalid audience or signature check locally: %v", err)
				return c.JSON(http.StatusUnauthorized, echo.Map{"message": "Invalid token audience"})
			}

			c.Set("userID", userID)
			c.Set("username", username)
			c.Set("role", role)

			return next(c)
		}
	}
}
