package api

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	db "github.com/LamThanhNguyen/banking-system/db/sqlc"
	"github.com/LamThanhNguyen/banking-system/token"
	"github.com/LamThanhNguyen/banking-system/util"
	"github.com/LamThanhNguyen/banking-system/worker"
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

// @Summary      Create a new user
// @Description  Register a new user and send email verification. Username and email must be unique.
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        body  body      createUserRequest  true  "User registration info"
// @Success      201   {object}  userResponse
// @Failure      400   {object}  api.ErrorResponse "Invalid request or validation error"
// @Failure      409   {object}  api.ErrorResponse "Conflict: email or username already exists"
// @Failure      500   {object}  api.ErrorResponse "Internal server error"
// @Router       /api/v1/users [post]
func (server *Server) createUser(ctx *gin.Context) {
	var req createUserRequest

	if !bindAndValidateJsonBody(ctx, &req) {
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
			ctx.JSON(http.StatusConflict, errorResponse(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	rsp := newUserResponse(txResult.User)
	ctx.JSON(http.StatusCreated, rsp)
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

// @Summary      Login
// @Description  Authenticate user and create a session with access & refresh tokens
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        body  body      loginUserRequest  true  "Login credentials"
// @Success      200   {object}  loginUserResponse
// @Failure      400   {object}  api.ErrorResponse "Invalid request or validation error"
// @Failure      401   {object}  api.ErrorResponse "Unauthorized: incorrect credentials"
// @Failure      404   {object}  api.ErrorResponse "User not found"
// @Failure      500   {object}  api.ErrorResponse "Internal server error"
// @Router       /api/v1/users/login [post]
func (server *Server) loginUser(ctx *gin.Context) {
	var req loginUserRequest
	if !bindAndValidateJsonBody(ctx, &req) {
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

type getUserRequest struct {
	Username string `uri:"username" binding:"required,username"`
}

type updateUserRequest struct {
	Password *string `json:"password,omitempty" binding:"omitempty,min=8,max=50"` // built-in tags
	FullName *string `json:"full_name,omitempty" binding:"omitempty,fullname"`    // custom tag
	Email    *string `json:"email,omitempty" binding:"omitempty,email,max=50"`    // built-in
}

// @Summary      Update user
// @Description  Banker can update any user. Depositor can update only their own account.
// @Tags         users
// @Security     BearerAuth
// @Param        username   path      string               true  "Username"
// @Param        body       body      updateUserRequest    true  "Fields to update"
// @Success      200        {object}  userResponse
// @Failure      400        {object}  api.ErrorResponse "Invalid request or validation error"
// @Failure      403        {object}  api.ErrorResponse "Forbidden: not allowed to update this user"
// @Failure      404        {object}  api.ErrorResponse "User not found"
// @Failure      409        {object}  api.ErrorResponse "Conflict: email or username already exists"
// @Failure      500        {object}  api.ErrorResponse "Internal server error"
// @Router       /api/v1/users/{username} [patch]
func (server *Server) updateUser(ctx *gin.Context) {
	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	var reqPath getUserRequest
	if err := ctx.ShouldBindUri(&reqPath); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var reqBody updateUserRequest
	if !bindAndValidateJsonBody(ctx, &reqBody) {
		return
	}

	if reqBody.Password == nil && reqBody.FullName == nil && reqBody.Email == nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(fmt.Errorf("no fields to update")))
		return
	}

	sub := util.Subject{
		Role: authPayload.Role,
		Name: authPayload.Username,
	}
	obj := util.Object{
		Name: reqPath.Username,
	}

	ok, err := server.enforcer.Enforce(sub, obj, "users:update")
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
	if !ok {
		ctx.JSON(http.StatusForbidden, errorResponse(fmt.Errorf("forbidden")))
		return
	}

	var fullName, email pgtype.Text
	if reqBody.FullName != nil {
		fullName = pgtype.Text{
			String: *reqBody.FullName, Valid: true,
		}
	}
	if reqBody.Email != nil {
		email = pgtype.Text{
			String: *reqBody.Email, Valid: true,
		}
	}

	arg := db.UpdateUserParams{
		Username: reqPath.Username,
		FullName: fullName,
		Email:    email,
	}

	if reqBody.Password != nil {
		hashedPassword, err := util.HashPassword(*reqBody.Password)
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
	} else {
		arg.PasswordChangedAt = pgtype.Timestamptz{}
	}

	user, err := server.store.UpdateUser(ctx, arg)
	if err != nil {
		switch {
		case errors.Is(err, db.ErrRecordNotFound):
			err := fmt.Errorf("user not found")
			ctx.JSON(http.StatusNotFound, errorResponse(err))
		case db.ErrorCode(err) == db.UniqueViolation:
			ctx.JSON(http.StatusConflict, errorResponse(err))
		default:
			ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		}
		return
	}

	rsp := newUserResponse(user)
	ctx.JSON(http.StatusOK, rsp)
}

type verifyEmailRequest struct {
	EmailId    int64  `form:"email_id" binding:"required,email_id"`
	SecretCode string `form:"secret_code" binding:"required,secret_code"`
}

// @Summary      Verify email
// @Description  Verify user email using a secret code sent via email
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        email_id    query     int    true   "Email ID"
// @Param        secret_code query     string true   "Secret verification code"
// @Success      204        "No Content: email verified successfully"
// @Failure      400        {object}  api.ErrorResponse "Invalid request or validation error"
// @Failure      404        {object}  api.ErrorResponse "Email or code not found"
// @Failure      500        {object}  api.ErrorResponse "Internal server error"
// @Router       /api/v1/users/verify_email [get]
func (server *Server) verifyEmail(ctx *gin.Context) {
	var req verifyEmailRequest

	if err := ctx.ShouldBindQuery(&req); err != nil {
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
			return
		}
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	if _, err := server.store.VerifyEmailTx(ctx, db.VerifyEmailTxParams{
		EmailId:    req.EmailId,
		SecretCode: req.SecretCode,
	}); err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			ctx.JSON(http.StatusNotFound, errorResponse(err))
		default:
			ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		}
		return
	}

	ctx.Status(http.StatusNoContent) // 204
}

func bindAndValidateJsonBody(ctx *gin.Context, v interface{}) bool {
	if err := ctx.ShouldBindJSON(v); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			violations := make([]FieldViolation, len(ve))
			for i, fe := range ve {
				violations[i] = FieldViolation{Field: fe.Field(), Message: humanMessage(fe)}
			}
			ctx.JSON(http.StatusBadRequest, gin.H{"violations": violations})
			return false
		}
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return false
	}
	return true
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
	case "email_id":
		return "must be a positive integer"
	default:
		return fe.Error() // fallback
	}
}
