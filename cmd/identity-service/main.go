package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mafi020/ecom-golang-micro/internal/bootstrap/identity"
)

func main() {
	app := identity.InitializeIdentityApp()
	srv, err := app.RunHTTP()
	if err != nil {
		log.Fatal(err)
	}

	grpcSrv, err := app.RunGRPC()
	if err != nil {
		log.Fatal(err)
	}

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	if err := app.ShutdownHTTP(srv); err != nil {
		log.Fatal(err)
	}

	if err := app.ShutdownGRPC(grpcSrv); err != nil {
		log.Fatal(err)
	}
}
