package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/lib/pq"
)

type Decision struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
}

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		fmt.Println("⚠️  Using local Docker database.")
		dbURL = "host=localhost port=5432 user=admin password=localpassword dbname=hinsight_local sslmode=disable"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("DB not responding. Did you run 'docker compose up -d'? Error: %v", err)
	}

	createTableQuery := `
	CREATE TABLE IF NOT EXISTS decisions (
		id SERIAL PRIMARY KEY,
		title TEXT NOT NULL
	);`
	_, err = db.Exec(createTableQuery)
	if err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}
	fmt.Println("✅ Database ready and 'decisions' table exists!")

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.POST("/decisions", func(c echo.Context) error {
		var req struct {
			Title string `json:"title"`
		}
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid JSON"})
		}

		var newID int
		err := db.QueryRow("INSERT INTO decisions (title) VALUES ($1) RETURNING id", req.Title).Scan(&newID)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to save to database"})
		}

		return c.JSON(http.StatusCreated, map[string]interface{}{
			"message": "Decision saved!",
			"id":      newID,
		})
	})

	e.GET("/decisions", func(c echo.Context) error {
		rows, err := db.Query("SELECT id, title FROM decisions")
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch from database"})
		}
		defer rows.Close()

		var decisions []Decision
		for rows.Next() {
			var d Decision
			if err := rows.Scan(&d.ID, &d.Title); err != nil {
				continue
			}
			decisions = append(decisions, d)
		}

		if decisions == nil {
			decisions = []Decision{}
		}

		return c.JSON(http.StatusOK, decisions)
	})

	e.Logger.Fatal(e.Start("0.0.0.0:8080"))
}
