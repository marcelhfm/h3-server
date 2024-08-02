package main

import (
	"log"
	"net/http"
	"time"

	"github.com/marcelhfm/h3-server/pkg/log"
)

func LoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		l.Log.Info().Msgf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
	})
}

func main() {
	l.Log.Info().Msg("Hello from h3 server!")
	router := http.NewServeMux()

	router.HandleFunc("POST /create-index", CreateIndexHandler())

	loggedRouter := LoggerMiddleware(router)
	log.Fatal(http.ListenAndServe(":5005", loggedRouter))
}
