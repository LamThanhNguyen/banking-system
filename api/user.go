package api

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	db "github.com/LamThanhNguyen/future-bank/db/sqlc"
	"github.com/LamThanhNguyen/future-bank/token"
	"github.com/LamThanhNguyen/future-bank/util"
	"github.com/LamThanhNguyen/future-bank/worker"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgtype"
)

type FieldViolation struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type createUserRequest struct {
	Username string `json:"username" binding:"required,username"`     // custom tag
	Password string `json:"password" binding:"required,min=8,max=50"` // built-in tags
	FullName string `json:"full_name" binding:"required,fullname"`    // custom tag
	Email    string `json:"email" binding:"required,email,max=50"`    // built-in
}

type userResponse struct {
	Username          string    `json:"username"`
	FullName          string    `json:"full_name"`
	Email             string    `json:"email"`
	PasswordChangedAt time.Time `json:"password_changed_at"`
	CreatedAt         time.Time `json:"created_at"`
}

func newUserResponse(user db.User) userResponse {
	return userResponse{
		Username:          user.Username,
		FullName:          user.FullName,
		Email:             user.Email,
		PasswordChangedAt: user.PasswordChangedAt,
		CreatedAt:         user.CreatedAt,
	}
}

func (server *Server) createUser(ctx *gin.Context) {
	var req createUserRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			violations := make([]FieldViolation, 0, len(ve))
			for _, fe := range ve {
				violations = append(violations, FieldViolation{
					Field:   fe.Field(),
					Message: humanMessage(fe),
				})
			}
			ctx.JSON(http.StatusBadRequest, gin.H{"violations": violations})
		}
		return
	}

	hashedPassword, err := util.HashPassword(req.Password)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	arg := db.CreateUserTxParams{
		CreateUserParams: db.CreateUserParams{
			Username:       req.Username,
			HashedPassword: hashedPassword,
			FullName:       req.FullName,
			Email:          req.Email,
		},
		AfterCreate: func(user db.User) error {
			taskPayload := &worker.PayloadSendVerifyEmail{
				Username: user.Username,
			}
			opts := []asynq.Option{
				asynq.MaxRetry(10),
				asynq.ProcessIn(10 * time.Second),
				asynq.Queue(worker.QueueCritical),
			}

			return server.taskDistributor.DistributeTaskSendVerifyEmail(ctx, taskPayload, opts...)
		},
	}

	txResult, err := server.store.CreateUserTx(ctx, arg)
	if err != nil {
		if db.ErrorCode(err) == db.UniqueViolation {
			ctx.JSON(http.StatusForbidden, errorResponse(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	rsp := newUserResponse(txResult.User)
	ctx.JSON(http.StatusOK, rsp)
}

type loginUserRequest struct {
	Username string `json:"username" binding:"required,username"`     // custom tag
	Password string `json:"password" binding:"required,min=8,max=50"` // built-in tags
}

type loginUserResponse struct {
	SessionID             uuid.UUID    `json:"session_id"`
	AccessToken           string       `json:"access_token"`
	AccessTokenExpiresAt  time.Time    `json:"access_token_expires_at"`
	RefreshToken          string       `json:"refresh_token"`
	RefreshTokenExpiresAt time.Time    `json:"refresh_token_expires_at"`
	User                  userResponse `json:"user"`
}

func (server *Server) loginUser(ctx *gin.Context) {
	var req loginUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			violations := make([]FieldViolation, 0, len(ve))
			for _, fe := range ve {
				violations = append(violations, FieldViolation{
					Field:   fe.Field(),
					Message: humanMessage(fe),
				})
			}
			ctx.JSON(http.StatusBadRequest, gin.H{"violations": violations})
		}
		return
	}

	user, err := server.store.GetUser(ctx, req.Username)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			err := fmt.Errorf("user not found")
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}
		err := fmt.Errorf("failed to find user")
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	err = util.CheckPassword(req.Password, user.HashedPassword)
	if err != nil {
		err := fmt.Errorf("incorrect password")
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	accessToken, accessPayload, err := server.tokenMaker.CreateToken(
		user.Username,
		user.Role,
		server.config.AccessTokenDuration,
		token.TokenTypeAccessToken,
	)
	if err != nil {
		err := fmt.Errorf("failed to create access token")
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	refreshToken, refreshPayload, err := server.tokenMaker.CreateToken(
		user.Username,
		user.Role,
		server.config.RefreshTokenDuration,
		token.TokenTypeRefreshToken,
	)
	if err != nil {
		err := fmt.Errorf("failed to create refresh token")
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	session, err := server.store.CreateSession(ctx, db.CreateSessionParams{
		ID:           refreshPayload.ID,
		Username:     user.Username,
		RefreshToken: refreshToken,
		UserAgent:    ctx.Request.UserAgent(),
		ClientIp:     ctx.ClientIP(),
		IsBlocked:    false,
		ExpiresAt:    refreshPayload.ExpiredAt,
	})
	if err != nil {
		err := fmt.Errorf("failed to create session")
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
	}

	rsp := loginUserResponse{
		SessionID:             session.ID,
		AccessToken:           accessToken,
		AccessTokenExpiresAt:  accessPayload.ExpiredAt,
		RefreshToken:          refreshToken,
		RefreshTokenExpiresAt: refreshPayload.ExpiredAt,
		User:                  newUserResponse(user),
	}
	ctx.JSON(http.StatusOK, rsp)
}

type updateUserRequest struct {
	Username string  `json:"username" binding:"required,username"`                // custom tag
	Password *string `json:"password,omitempty" binding:"omitempty,min=8,max=50"` // built-in tags
	FullName *string `json:"full_name,omitempty" binding:"omitempty,fullname"`    // custom tag
	Email    *string `json:"email,omitempty" binding:"omitempty,email,max=50"`    // built-in
}

func (server *Server) updateUser(ctx *gin.Context) {
	authPayload := ctx.MustGet(authorizationHeaderKey).(*token.Payload)

	var req updateUserRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			violations := make([]FieldViolation, 0, len(ve))
			for _, fe := range ve {
				violations = append(violations, FieldViolation{
					Field:   fe.Field(),
					Message: humanMessage(fe),
				})
			}
			ctx.JSON(http.StatusBadRequest, gin.H{"violations": violations})
		}
		return
	}

	// banker role can update other user's info
	// depositor only update their's info
	if authPayload.Role != util.BankerRole && authPayload.Username != req.Username {
		err := fmt.Errorf("depositor cannot update other user's info")
		ctx.JSON(http.StatusForbidden, errorResponse(err))
		return
	}

	arg := db.UpdateUserParams{
		Username: req.Username,
		FullName: pgtype.Text{
			String: *req.FullName,
			Valid:  req.FullName != nil,
		},
		Email: pgtype.Text{
			String: *req.Email,
			Valid:  req.Email != nil,
		},
	}

	if req.Password != nil {
		hashedPassword, err := util.HashPassword(*req.Password)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, errorResponse(err))
			return
		}

		arg.HashedPassword = pgtype.Text{
			String: hashedPassword,
			Valid:  true,
		}

		arg.PasswordChangedAt = pgtype.Timestamptz{
			Time:  time.Now(),
			Valid: true,
		}
	}

	user, err := server.store.UpdateUser(ctx, arg)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			err := fmt.Errorf("user not found")
			ctx.JSON(http.StatusNotFound, errorResponse(err))
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
	}

	rsp := newUserResponse(user)
	ctx.JSON(http.StatusOK, rsp)
}

func humanMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "is required"
	case "min":
		return "must be at least " + fe.Param() + " characters"
	case "max":
		return "must be at most " + fe.Param() + " characters"
	case "username":
		return "must contain only lowercase letters, digits or underscore"
	case "fullname":
		return "must contain only letters or spaces"
	case "email":
		return "is not a valid email address"
	default:
		return fe.Error() // fallback
	}
}
