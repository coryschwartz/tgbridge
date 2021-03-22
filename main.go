package main

import (
	"net/http"
	"time"

	"github.com/gregjones/httpcache"
	"github.com/palantir/go-githubapp/githubapp"
)

func main() {
	config, err := ReadConfig("config.yml")
	if err != nil {
		panic(err)
	}

	cc, err := githubapp.NewDefaultCachingClientCreator(
		config.Github,
		githubapp.WithClientUserAgent("tgbridge/0.0.0"),
		githubapp.WithClientTimeout(3*time.Second),
		githubapp.WithClientCaching(false, func() httpcache.Cache { return httpcache.NewMemoryCache() }),
	)
	if err != nil {
		panic(err)
	}

	webhookHandler := githubapp.NewDefaultEventDispatcher(config.Github,
		&CheckHandler{
			ClientCreator: cc,
		},
	)

	server := http.Server{
		Addr:    config.Listen,
		Handler: webhookHandler,
	}
	err = server.ListenAndServe()
	if err != nil {
		panic(err)
	}
}
