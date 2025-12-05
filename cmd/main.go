package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		fmt.Println("Error: DB_URL environment variable is not set")
		os.Exit(1)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "4001"
	}

	ctx := context.Background()
	fmt.Println("Connecting to database...")
	db, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		fmt.Println("failed to connect to postgres:", err)
		os.Exit(1)
	}
	defer db.Close()
	fmt.Println("Database connection established.")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "service running")
	})

	server := &http.Server{
		Addr:         ":" + port,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		fmt.Printf("Server starting on port %s...\n", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Println("server error:", err)
			os.Exit(1)
		}
	}()

	time.Sleep(100 * time.Millisecond)
	fmt.Println("Server is running. Press Ctrl+C to stop.")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	fmt.Println("\nShutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(shutdownCtx)
}