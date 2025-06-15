package api

import (
	"fmt"
	"net/http"
	"time"

	db "github.com/LamThanhNguyen/banking-system/db/sqlc"
	"github.com/LamThanhNguyen/banking-system/token"
	"github.com/LamThanhNguyen/banking-system/util"
	"github.com/LamThanhNguyen/banking-system/worker"
	"github.com/casbin/casbin/v2"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type Server struct {
	config          util.Config
	store           db.Store
	enforcer        *casbin.Enforcer
	router          *gin.Engine
	tokenMaker      token.Maker
	taskDistributor worker.TaskDistributor
}

func NewServer(
	config util.Config,
	store db.Store,
	enforcer *casbin.Enforcer,
	taskDistributor worker.TaskDistributor,
) (*Server, error) {
	tokenMaker, err := token.NewPasetoMaker(config.TokenSymmetricKey)
	if err != nil {
		return nil, fmt.Errorf("cannot create token maker: %w", err)
	}

	return &Server{
		config:          config,
		store:           store,
		enforcer:        enforcer,
		tokenMaker:      tokenMaker,
		taskDistributor: taskDistributor,
	}, nil
}

func (server *Server) SetupRouter() {
	const requestTimeout = 30 * time.Second
	if server.config.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	router := gin.New()

	corsCfg := cors.Config{
		AllowOrigins:     server.config.AllowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}

	// CORS middleware
	router.Use(
		gin.Recovery(),
		HttpLogger(),
		cors.New(corsCfg),
		timeoutMiddleware(requestTimeout),
	)

	if server.config.Environment == "development" {
		router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	apiRoutes := router.Group("/api/v1")
	{
		apiRoutes.GET("/health", server.handleHealthCheck)
		apiRoutes.POST("/users", server.createUser)
		apiRoutes.POST("/users/login", server.loginUser)
		apiRoutes.POST("/tokens/renew-access", server.renewAccessToken)
		apiRoutes.GET("/users/verify-email", server.verifyEmail)

		authRoutes := apiRoutes.Group("", authMiddleware(server.tokenMaker))
		authRoutes.PATCH(
			"/users/:username",
			server.Require("users:update"),
			server.updateUser,
		)
		authRoutes.POST(
			"/accounts",
			server.Require("accounts:create"),
			server.createAccount,
		)
		authRoutes.GET(
			"/accounts/:id",
			server.Require("accounts:read"),
			server.getAccount,
		)
		authRoutes.GET(
			"/accounts",
			server.Require("accounts:list"),
			server.listAccounts,
		)
		authRoutes.POST(
			"/transfers",
			server.Require("transfers:create"),
			server.createTransfer,
		)
	}

	server.router = router
}

func (server *Server) Router() *gin.Engine {
	return server.router
}

func (server *Server) Start(address string) error {
	return server.router.Run(address)
}

func (server *Server) handleHealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// ErrorResponse defines the standard error response for the API.
type ErrorResponse struct {
	Error      string           `json:"error"`
	Violations []FieldViolation `json:"violations,omitempty"`
}

func errorResponse(err error) gin.H {
	return gin.H{"error": err.Error()}
}
