package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/vporoshok/di/example/config"
	"github.com/vporoshok/di/example/provider"
	httpTransport "github.com/vporoshok/di/example/transport/http"

	"github.com/vporoshok/di"
)

func main() {
	ctx := context.Background()
	dc, err := makeDC(ctx)
	if err != nil {
		log.Fatal(err)
	}
	httpServer := dc.MustMake(ctx, httpTransport.MakeServer).(*http.Server)
	go httpServer.ListenAndServe()
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	<-c
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err = httpServer.Shutdown(ctx); err != nil {
		log.Fatal(err)
	}
}

func makeDC(ctx context.Context) (di.Container, error) {
	dc := di.NewContainer()
	dc.RegisterFunc("config", config.MakeConfig)
	dc.RegisterStruct("cryptoProvider", provider.Crypto)
	return dc, dc.Check(ctx)
	// var act interface {
	// 	Do(email, name string) error
	// }
	// if err := dc.Get(context.Background(), "create user", &act); err != nil {
	// 	log.Fatal(err)
	// }
	// if err := act.Do("me@example.com", "test"); err != nil {
	// 	log.Fatal(err)
	// }
	// log.Print("users added")
	// if err := dc.Get(context.Background(), "create user", &act); err != nil {
	// 	log.Fatal(err)
	// }
	// if err := act.Do("me@example.com", "test"); err != nil {
	// 	log.Fatal(err)
	// }
	// log.Print("users added")
	// t := &GetUser{}
	// if err := dc.Get(context.Background(), "get user", &t); err != nil {
	// 	log.Fatal(err)
	// }
}
