package auth

import (
	"errors"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID int64 `json:"uid"`
	jwt.RegisteredClaims
}

func Sign(secret string, userID int64, ttl time.Duration, now time.Time) (string, error) {
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(userID, 10),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func Parse(secret, raw string) (int64, error) {
	claims := &Claims{}
	parsed, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("허용하지 않는 서명 방식")
		}
		return []byte(secret), nil
	})
	if err != nil || !parsed.Valid || claims.UserID < 1 {
		if err == nil {
			err = errors.New("토큰이 유효하지 않습니다")
		}
		return 0, err
	}
	return claims.UserID, nil
}
