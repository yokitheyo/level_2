package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

func main() {
	port := flag.String("port", "8080", "HTTP server port")
	flag.Parse()

	calendar := NewCalendar()
	handler := NewHandler(calendar)

	router := mux.NewRouter()
	router.Use(LoggingMiddleware)
	handler.RegisterRoutes(router)

	log.Printf("Starting server on port %s", *port)
	if err := http.ListenAndServe(":"+*port, router); err != nil {
		log.Fatalf("Could not start server: %v", err)
		os.Exit(1)
	}
}
