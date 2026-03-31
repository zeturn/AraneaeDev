package control

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

type authClaims struct {
	UserID string `json:"uid"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func (a *App) issueToken(userID, role string) (string, error) {
	claims := authClaims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(12 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(a.cfg.JWTSecret))
}

func (a *App) parseToken(raw string) (*authClaims, error) {
	tok, err := jwt.ParseWithClaims(raw, &authClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected jwt signing method")
		}
		return []byte(a.cfg.JWTSecret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := tok.Claims.(*authClaims)
	if !ok || !tok.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

func (a *App) authMiddleware(c *fiber.Ctx) error {
	header := c.Get("Authorization")
	if len(header) < 8 || header[:7] != "Bearer " {
		return fiber.NewError(fiber.StatusUnauthorized, "missing bearer token")
	}
	claims, err := a.parseToken(header[7:])
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid bearer token")
	}
	c.Locals("uid", claims.UserID)
	c.Locals("role", claims.Role)
	return c.Next()
}

func (a *App) requireRoles(roles ...string) fiber.Handler {
	allowed := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}
	return func(c *fiber.Ctx) error {
		role, _ := c.Locals("role").(string)
		if _, ok := allowed[role]; !ok {
			return fiber.NewError(fiber.StatusForbidden, "insufficient permissions")
		}
		return c.Next()
	}
}
