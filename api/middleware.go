package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/LamThanhNguyen/future-bank/token"
	"github.com/LamThanhNguyen/future-bank/util"
	"github.com/gin-gonic/gin"
)

const (
	authorizationHeaderKey  = "authorization"
	authorizationTypeBearer = "bearer"
	authorizationPayloadKey = "authorization_payload"
)

func authMiddleware(tokenMaker token.Maker) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authorizationHeader := ctx.GetHeader(authorizationHeaderKey) // authorization

		if len(authorizationHeader) == 0 {
			err := errors.New("authorization header is not provided")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		fields := strings.Fields(authorizationHeader)
		if len(fields) < 2 {
			err := errors.New("invalid authorization header format")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		authorizationType := strings.ToLower(fields[0])
		if authorizationType != authorizationTypeBearer {
			// not bearer
			err := fmt.Errorf("unsupported authorization type %s", authorizationType)
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		accessToken := fields[1]
		payload, err := tokenMaker.VerifyToken(accessToken, token.TokenTypeAccessToken)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		ctx.Set(authorizationPayloadKey, payload)
		ctx.Next()
	}
}

func (s *Server) Require(action string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		p, ok := ctx.Get(authorizationPayloadKey)

		if !ok {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized,
				gin.H{"error": "missing auth payload"},
			)
			return
		}

		payload := p.(*token.Payload)

		sub := util.Subject{Role: payload.Role, Name: payload.Username}
		obj := util.Object{Name: "*"} // wildcard for everything except ABAC handlers

		allowed, err := s.enforcer.Enforce(sub, obj, action)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if !allowed {
			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}

		ctx.Next()
	}
}
