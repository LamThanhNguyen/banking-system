package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/LamThanhNguyen/banking-system/api"
	db "github.com/LamThanhNguyen/banking-system/db/sqlc"
	_ "github.com/LamThanhNguyen/banking-system/docs" // swagger docs init
	"github.com/LamThanhNguyen/banking-system/mail"
	pgxadapter "github.com/LamThanhNguyen/banking-system/pgxadapter"
	"github.com/LamThanhNguyen/banking-system/util"
	"github.com/LamThanhNguyen/banking-system/worker"
	"github.com/casbin/casbin/v2"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

var interruptSignals = []os.Signal{
	os.Interrupt,
	syscall.SIGTERM,
	syscall.SIGINT,
}

// @title           Be Banking System API
// @version         1.0
// @description     API documentation for the Be Banking System project.
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token. Example: "Bearer <token>"
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), interruptSignals...)
	defer stop()

	config, err := util.LoadConfig(ctx, ".")
	if err != nil {
		log.Fatal().Err(err).Msg("cannot load config")
	}

	log.Info().Interface("config", config).Msg("loaded config")

	connPool, err := pgxpool.New(ctx, config.DBSource)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot connect to db")
	}

	// Run migration to database
	runDBMigration(config.MigrationURL, config.DBSource)

	casbin_adapter, err := pgxadapter.New(ctx, connPool)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot create casbin adapter")
	}

	casbin_enforcer, err := casbin.NewEnforcer("model.conf", casbin_adapter)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot create casbin enforcer")
	}
	casbin_enforcer.EnableAutoSave(true)

	if err := seedPolicies(casbin_enforcer); err != nil {
		log.Fatal().Err(err).Msg("cannot seed Policies")
	}

	store := db.NewStore(connPool)

	redisOpt := asynq.RedisClientOpt{
		Addr: config.RedisAddress,
	}

	taskDistributor := worker.NewRedisTaskDistributor(redisOpt)

	waitGroup, ctx := errgroup.WithContext(ctx)

	runTaskProcessor(ctx, waitGroup, config, redisOpt, store)
	runServer(ctx, waitGroup, config, store, casbin_enforcer, taskDistributor)

	if err = waitGroup.Wait(); err != nil {
		log.Fatal().Err(err).Msg("err from wait group")
	}

	log.Info().Msg("application shutdown complete")
}

func runDBMigration(migrationURL string, dbSource string) {
	migration, err := migrate.New(migrationURL, dbSource)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot create new migrate instance")
	}

	if err = migration.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatal().Err(err).Msg("failed to run migrate up")
	}

	log.Info().Msg("db migrated successfully")
}

func seedPolicies(casbin_enforcer *casbin.Enforcer) error {
	add := func(sub, act string) {
		_, _ = casbin_enforcer.AddPolicy(sub, "*", act)
	}

	// banker
	add("banker", "accounts:create")
	add("banker", "accounts:read")
	add("banker", "accounts:list")
	add("banker", "users:update")
	add("banker", "transfers:create")

	// depositer
	add("depositor", "accounts:create")
	add("depositor", "accounts:read")
	add("depositor", "accounts:list")
	add("depositor", "users:update")
	add("depositor", "transfers:create")
	return nil
}

func runTaskProcessor(
	ctx context.Context,
	waitGroup *errgroup.Group,
	config util.Config,
	redisOpt asynq.RedisClientOpt,
	store db.Store,
) {
	mailer := mail.NewGmailSender(config.EmailSenderName, config.EmailSenderAddress, config.EmailSenderPassword)
	taskProcessor := worker.NewRedisTaskProcessor(redisOpt, store, mailer)

	if err := taskProcessor.Start(); err != nil {
		log.Fatal().Err(err).Msg("failed to start task processor")
	}
	log.Info().Msg("task processor started")

	waitGroup.Go(func() error {
		<-ctx.Done()
		log.Info().Msg("graceful shutdown task processor")

		taskProcessor.Shutdown()
		log.Info().Msg("task processor is stopped")

		return nil
	})
}

func runServer(
	ctx context.Context,
	waitGroup *errgroup.Group,
	config util.Config,
	store db.Store,
	enforcer *casbin.Enforcer,
	taskDistributor worker.TaskDistributor,
) {
	server, err := api.NewServer(config, store, enforcer, taskDistributor)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot create server")
	}

	server.SetupRouter() // initialize routes

	// Setup HTTP server
	httpServer := &http.Server{
		Addr:    config.HTTPServerAddress,
		Handler: server.Router(), // use the Gin router
	}

	// Start HTTP server in goroutine
	waitGroup.Go(func() error {
		log.Info().Msgf("start HTTP server at %s", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error().Err(err).Msg("HTTP server failed to serve")
			return err
		}
		return nil
	})

	// Graceful shutdown on context cancel
	waitGroup.Go(func() error {
		<-ctx.Done()
		log.Info().Msg("graceful shutdown HTTP server")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.Error().Err(err).Msg("failed to shutdown HTTP server")
			return err
		}

		log.Info().Msg("HTTP server is stopped")
		return nil
	})
}
