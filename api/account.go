package api

import (
	"errors"
	"net/http"

	db "github.com/LamThanhNguyen/banking-system/db/sqlc"
	"github.com/LamThanhNguyen/banking-system/token"
	"github.com/gin-gonic/gin"
)

type createAccountRequest struct {
	Currency string `json:"currency" binding:"required,currency"`
}

// @Summary      Create account
// @Description  Create a new bank account for the authenticated user
// @Tags         accounts
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body  body      createAccountRequest  true  "Account info"
// @Success      200   {object}  db.Account
// @Failure      400   {object}  api.ErrorResponse "Invalid request or validation error"
// @Failure      403   {object}  api.ErrorResponse "Forbidden: account already exists or invalid foreign key"
// @Failure      500   {object}  api.ErrorResponse "Internal server error"
// @Router       /api/v1/accounts [post]
func (server *Server) createAccount(ctx *gin.Context) {
	var req createAccountRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	arg := db.CreateAccountParams{
		Owner:    authPayload.Username,
		Currency: req.Currency,
		Balance:  0,
	}

	account, err := server.store.CreateAccount(ctx, arg)
	if err != nil {
		errCode := db.ErrorCode(err)
		if errCode == db.ForeignKeyViolation || errCode == db.UniqueViolation {
			ctx.JSON(http.StatusForbidden, errorResponse(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, account)
}

type getAccountRequest struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

// @Summary      Get account
// @Description  Get an account by its ID. Only the owner can access their account.
// @Tags         accounts
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      int  true  "Account ID"
// @Success      200  {object}  db.Account
// @Failure      400  {object}  api.ErrorResponse "Invalid request"
// @Failure      401  {object}  api.ErrorResponse "Unauthorized: not account owner"
// @Failure      404  {object}  api.ErrorResponse "Account not found"
// @Failure      500  {object}  api.ErrorResponse "Internal server error"
// @Router       /api/v1/accounts/{id} [get]
func (server *Server) getAccount(ctx *gin.Context) {
	var req getAccountRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	account, err := server.store.GetAccount(ctx, req.ID)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	if account.Owner != authPayload.Username {
		err := errors.New("account doesn't belong to the authenticated user")
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, account)
}

type listAccountRequest struct {
	PageID   int32 `form:"page_id" binding:"required,min=1"`
	PageSize int32 `form:"page_size" binding:"required,min=5,max=10"`
}

// @Summary      List accounts
// @Description  List all accounts for the authenticated user (paginated)
// @Tags         accounts
// @Security     BearerAuth
// @Produce      json
// @Param        page_id   query     int  true  "Page number (min 1)"
// @Param        page_size query     int  true  "Page size (min 5, max 10)"
// @Success      200       {array}   db.Account
// @Failure      400       {object}  api.ErrorResponse "Invalid request"
// @Failure      500       {object}  api.ErrorResponse "Internal server error"
// @Router       /api/v1/accounts [get]
func (server *Server) listAccounts(ctx *gin.Context) {
	var req listAccountRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	arg := db.ListAccountsParams{
		Owner:  authPayload.Username,
		Limit:  req.PageSize,
		Offset: (req.PageID - 1) * req.PageSize,
	}

	accounts, err := server.store.ListAccounts(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, accounts)
}
