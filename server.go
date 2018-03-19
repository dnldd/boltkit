package main

import (
	"context"
	"einheit/boltkit/scheduler"
	"einheit/boltkit/service"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/gorilla/handlers"
)

func main() {
	var err error

	// Initialize log rotation.
	initLogRotator(filepath.Join("log", "server.log"))

	// Initialize application.
	service.App, err = service.NewService("config.json")
	if err != nil {
		log.Error(err)
	}

	// Initialize the http server.
	service.App.SetupRoutes()
	service.Server = http.Server{
		Addr:    service.App.Cfg.Port,
		Handler: handlers.CORS()(service.App.Router),
	}

	// Initialize the job scheduler.
	scheduler.AppScheduler = scheduler.NewScheduler()
	scheduler.AppScheduler.Schedule(service.App)

	idleConnsClosed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint

		// Teardown service.
		// Save all sessions to the session store before shutdown.
		service.App.SaveSessions()
		service.App.Bolt.Close()
		log.Info("Shutdown complete.")

		// Received an interrupt signal, shut down.
		if err := service.Server.Shutdown(context.Background()); err != nil {
			log.Errorf("Failed to shutdown server: %v", err)
		}
		close(idleConnsClosed)
	}()

	// Start the server.
	log.Infof("Starting %s on port %s", service.App.Cfg.Server, service.App.Cfg.Port)
	if service.App.Cfg.HTTPS {
		if err := service.Server.ListenAndServeTLS("cert.pem", "privkey.pem"); err != http.ErrServerClosed {
			log.Error(err)
		}
	} else {
		if err := service.Server.ListenAndServe(); err != http.ErrServerClosed {
			log.Error(err)
		}
	}

	<-idleConnsClosed
}
