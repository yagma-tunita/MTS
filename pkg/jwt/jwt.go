package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// CustomClaims defines the structure of JWT claims.
type CustomClaims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"` // "shipper", "shipping", "admin"
	jwt.RegisteredClaims
}

// JWTService defines the interface for JWT operations.
type JWTService interface {
	GenerateAccessToken(userID int64, username, role string) (string, error)
	GenerateRefreshToken(userID int64, username, role string) (string, error)
	ValidateToken(tokenString string) (*CustomClaims, error)
	RefreshAccessToken(refreshTokenString string) (string, error)
}

type jwtService struct {
	secret        []byte
	accessExpire  time.Duration
	refreshExpire time.Duration
}

// NewJWTService creates a new JWT service with given secret and expiration times.
func NewJWTService(secret string, accessExpire, refreshExpire time.Duration) JWTService {
	return &jwtService{
		secret:        []byte(secret),
		accessExpire:  accessExpire,
		refreshExpire: refreshExpire,
	}
}

// GenerateAccessToken generates a short-lived access token.
func (s *jwtService) GenerateAccessToken(userID int64, username, role string) (string, error) {
	claims := &CustomClaims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.accessExpire)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

// GenerateRefreshToken generates a long-lived refresh token.
func (s *jwtService) GenerateRefreshToken(userID int64, username, role string) (string, error) {
	claims := &CustomClaims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.refreshExpire)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

// ValidateToken validates the token and returns the custom claims.
func (s *jwtService) ValidateToken(tokenString string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.secret, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, errors.New("token expired")
		}
		if errors.Is(err, jwt.ErrTokenSignatureInvalid) {
			return nil, errors.New("invalid token signature")
		}
		return nil, errors.New("invalid token")
	}
	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}
	return claims, nil
}

// RefreshAccessToken validates the refresh token and returns a new access token.
// It does NOT issue a new refresh token (to keep rotation simple). If you need a new refresh token as well,
// you can extend this function.
func (s *jwtService) RefreshAccessToken(refreshTokenString string) (string, error) {
	claims, err := s.ValidateToken(refreshTokenString)
	if err != nil {
		return "", err
	}
	// Ensure the token is a refresh token by checking its expiration (optional: could check claim type)
	// For simplicity, we allow any valid token to be used for refresh, but typically you'd want to ensure
	// the token has a longer expiration. Here we just generate a new access token.
	return s.GenerateAccessToken(claims.UserID, claims.Username, claims.Role)
}
