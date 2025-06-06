package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strconv"
	"sync"
	"time"

	"github.com/musannif-md/musannif/internal/config"
	"github.com/musannif-md/musannif/internal/db"
	"github.com/musannif-md/musannif/internal/logger"
	"github.com/musannif-md/musannif/internal/middlewares"
	"github.com/musannif-md/musannif/internal/routes"
	"github.com/musannif-md/musannif/internal/utils"
)

func initialize() error {
	err := config.Initialize()
	if err != nil {
		return fmt.Errorf("error initializing config: %v\n", err)
	}

	err = logger.Initialize(logger.LoggerConfig{
		InfoLogPath:  path.Join(config.Cfg.App.LogDirectory, "info.log"),
		ErrorLogPath: path.Join(config.Cfg.App.LogDirectory, "error.log"),
	})
	if err != nil {
		log.Fatalf("error initializing logger: %v\n", err)
	}

	utils.SetJwtKeys(config.Cfg.Secrets.JWT_ACCESS_SECRET, config.Cfg.Secrets.JWT_REFRESH_SECRET)

	err = db.InitDb(config.Cfg.App.SqliteDirectory)
	if err != nil {
		log.Fatalf("error initializing db: %v\n", err)
	}

	return nil
}

func main() {
	err := initialize()
	if err != nil {
		log.Fatalf("error during init: %v\n", err)
	}

	userSignup := flag.Bool("signup", false, "Create user")
	serve := flag.Bool("serve", false, "Start server")
	username := flag.String("username", "", "Username for user")
	password := flag.String("password", "", "Password for user")
	flag.Parse()

	if !*userSignup && !*serve {
		fmt.Fprintln(os.Stderr, "No command specified. Use `--signup` or `-serve`")
		os.Exit(1)
	}

	if *userSignup {
		if *username == "" || *password == "" {
			log.Fatal("username and password are required for signup")
			os.Exit(1)
		}

		err := db.SignupUser(*username, *password, "user")
		if err != nil {
			log.Fatalf("error creating user: %v\n", err)
			os.Exit(1)
		}

		return
	}

	if *serve {
		ctx := context.Background()
		err := run(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	}
}

func run(ctx context.Context) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	defer func() {
		err := db.CleanupDb()
		if err != nil {
			logger.Log.Error().Err(err).Msg("Failed to cleanup database")
		}
	}()

	srv := newServer(&config.Cfg)

	httpServer := &http.Server{
		Addr:    net.JoinHostPort(config.Cfg.Server.Host, strconv.Itoa(config.Cfg.Server.Port)),
		Handler: srv,
	}

	go func() {
		logger.Log.Printf("listening on %s\n", httpServer.Addr)
		err := httpServer.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			logger.Log.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			logger.Log.Error().Err(err).Msg("Failed to shutdown server")
		}
	}()

	wg.Wait()

	return nil
}

func newServer(cfg *config.AppConfig) http.Handler {
	mux := http.NewServeMux()

	routes.AddRoutes(mux, cfg)

	var handler http.Handler = middlewares.CORS(mux)

	if cfg.App.Environment == "debug" {
		handler = middlewares.ConditionalLogger(handler)
	}

	return handler
}
