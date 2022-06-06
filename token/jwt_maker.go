package token

import (
	"errors"
	"fmt"
	"time"

	"github.com/dgrijalva/jwt-go"
)

const minSecretKeySize = 32

// JWTMaker is a JSON Web Token maker
type JWTMaker struct {
	secretKey string
}

// NewJWTMaker creates a new JWTMaker
func NewJWTMaker(secretKey string) (Maker, error) {
	if len(secretKey) < minSecretKeySize {
		return nil, fmt.Errorf("invalid key size: must be at least %d characters", minSecretKeySize)
	}

	return &JWTMaker{secretKey}, nil
}

// CreateToken creates a new token for a specific username and duration
func (maker *JWTMaker) CreateToken(
	username string,
	duration time.Duration,
) (string, *Payload, error) {
	payload, err := NewPayload(username, duration)
	if err != nil {
		return "", payload, err
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)
	token, err := jwtToken.SignedString([]byte(maker.secretKey))
	return token, payload, err
}

// VerifyToken checks if the token is valid or not
func (maker *JWTMaker) VerifyToken(token string) (*Payload, error) {
	// callback function that receives the passed, but unverified token. Verify
	// the header to make sure that the signing algorithm matches.
	keyFunc := func(token *jwt.Token) (interface{}, error) {
		// get signing algorithm. Converting it to HMAC since we are using HS256
		_, ok := token.Method.(*jwt.SigningMethodHMAC)
		if !ok {
			// algorithm doesn't match with our signing algorithm
			return nil, ErrInvalidToken
		}
		// algorithm matches
		return []byte(maker.secretKey), nil
	}
	// Parse token
	jwtToken, err := jwt.ParseWithClaims(token, &Payload{}, keyFunc)
	if err != nil {
		verr, ok := err.(*jwt.ValidationError)
		if ok && errors.Is(verr.Inner, ErrExpiredToken) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}
	// after it's successfully parsed and verified, get payload data
	payload, ok := jwtToken.Claims.(*Payload)
	if !ok {
		return nil, ErrInvalidToken
	}
	return payload, nil
}
