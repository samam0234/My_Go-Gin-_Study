package app

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/samam0234/my-go-gin-study/internal/config"
	"github.com/samam0234/my-go-gin-study/internal/httpapi"
	"github.com/samam0234/my-go-gin-study/internal/store"
)

func Run() error {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		return err
	}
	st, err := store.Open(cfg.Driver, cfg.DSN)
	if err != nil {
		return err
	}
	defer st.Close()
	if err := st.Migrate(context.Background()); err != nil {
		return err
	}
	srv := httpapi.New(cfg, st)
	if cfg.Seed {
		if err := srv.Seed(context.Background()); err != nil {
			return err
		}
	}
	httpServer := &http.Server{
		Addr:              cfg.Addr,
		Handler:           srv.Router(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go srv.RunMailer(ctx)
	go func() {
		<-ctx.Done()
		shut, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shut)
	}()
	log.Printf("gin study listening on %s env=%s db=%s", cfg.Addr, cfg.Env, cfg.Driver)
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
