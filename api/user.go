package api

import (
	"errors"
	"net/http"
	"time"

	db "github.com/LamThanhNguyen/future-bank/db/sqlc"
	"github.com/LamThanhNguyen/future-bank/util"
	"github.com/LamThanhNguyen/future-bank/worker"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/hibiken/asynq"
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
