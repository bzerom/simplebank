package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type rewnewAccessTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type renewAccessTokenResponse struct {
	AccessToken          string    `json:"access_token"`
	AccessTokenExpiresAt time.Time `json:"access_token_expires_at"`
}

func (server *Server) renewAccessToken(ctx *gin.Context) {
	var req rewnewAccessTokenRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// One of the benefit of using paseto is if an invalid token is used, the
	// server doesn't have to query the database at all. It will just result in
	// "token is invalid"
	refreshPayload, err := server.tokenMaker.VerifyToken(req.RefreshToken)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	session, err := server.store.GetSession(ctx, refreshPayload.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	if session.IsBlocked {
		err := fmt.Errorf("blocked session")
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	if session.Username != refreshPayload.Username {
		err := fmt.Errorf("incorrect session user")
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	if session.RefreshToken != req.RefreshToken {
		err := fmt.Errorf("mismatched session")
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	// Even-though the expiration time is checked in  verifyToken, we might need
	// to check it again since in some rare cases, we might want to force it to
	// expire before it's actual expiration time.
	if time.Now().After(session.ExpiresAt) {
		err := fmt.Errorf("expires session")
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	accessToken, accessPayload, err := server.tokenMaker.CreateToken(
		refreshPayload.Username,
		server.config.RefreshTokenDuration,
	)

	rsp := renewAccessTokenResponse{
		AccessToken:          accessToken,
		AccessTokenExpiresAt: accessPayload.ExpiredAt,
	}
	ctx.JSON(http.StatusOK, rsp)
}
