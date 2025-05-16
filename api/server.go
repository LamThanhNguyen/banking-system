package api

import (
	"fmt"
	"net/http"
	"time"

	db "github.com/LamThanhNguyen/future-bank/db/sqlc"
	"github.com/LamThanhNguyen/future-bank/token"
	"github.com/LamThanhNguyen/future-bank/util"
	"github.com/LamThanhNguyen/future-bank/worker"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type Server struct {
	config          util.Config
	store           db.Store
	tokenMaker      token.Maker
	router          *gin.Engine
	taskDistributor worker.TaskDistributor
}

func NewServer(config util.Config, store db.Store, taskDistributor worker.TaskDistributor) (*Server, error) {
	tokenMaker, err := token.NewPasetoMaker(config.TokenSymmetricKey)
	if err != nil {
		return nil, fmt.Errorf("cannot create token maker: %w", err)
	}

	return &Server{
		config:          config,
		store:           store,
		tokenMaker:      tokenMaker,
		taskDistributor: taskDistributor,
	}, nil
}

func (server *Server) SetupRouter() {
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
	)

	router.Static("/swagger", "./swagger")

	apiRoutes := router.Group("/api/v1")
	{
		apiRoutes.GET("/health", server.handleHealthCheck)
		apiRoutes.POST("/users", server.createUser)
		apiRoutes.POST("/users/login", server.loginUser)
		apiRoutes.POST("/tokens/renew_access", server.renewAccessToken)

		authRoutes := apiRoutes.Group("", authMiddleware(server.tokenMaker))
		authRoutes.POST("/accounts", server.createAccount)
		authRoutes.GET("/accounts/:id", server.getAccount)
		authRoutes.GET("/accounts", server.listAccounts)
		authRoutes.POST("/transfers", server.createTransfer)
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

func errorResponse(err error) gin.H {
	return gin.H{"error": err.Error()}
}
