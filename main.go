package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Bookmark represents a bookmark with an ID, title, and URL.
type Bookmark struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title" binding:"required"`
	URL       string    `json:"url" binding:"required,url"`
	CreatedAt time.Time `json:"created_at"`
}

// Database connection pool, created once in main()
var db *pgxpool.Pool

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("Database URL not set")
	}

	startupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var err error

	db, err = pgxpool.New(startupCtx, dbURL)
	if err != nil {
		log.Fatalf("Unable to create connection pool %v", err)
	}
	defer db.Close()

	if err := db.Ping(startupCtx); err != nil {
		log.Fatalf("Unable to reach database: %v", err)
	}
	log.Printf("connected to postgres")

	router := gin.Default()

	// Read all bookmarks
	router.GET("/bookmarks", func(ctx *gin.Context) {
		rows, err := db.Query(ctx.Request.Context(),
			"SELECT id, title, url, created_at FROM bookmarks ORDER BY id")
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
			return
		}
		defer rows.Close()

		// Empty slice (not nil)
		bookmarks := []Bookmark{}
		for rows.Next() {
			var b Bookmark
			if err := rows.Scan(&b.ID, &b.Title, &b.URL, &b.CreatedAt); err != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
				return
			}
			bookmarks = append(bookmarks, b)
		}
		if err := rows.Err(); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
			return
		}
		ctx.JSON(http.StatusOK, bookmarks)
	})

	// Read one bookmark by ID
	router.GET("/bookmarks/:id", func(ctx *gin.Context) {
		id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
			return
		}

		var b Bookmark
		err = db.QueryRow(ctx.Request.Context(),
			"SELECT id, title, url, created_at FROM bookmarks WHERE id = $1", id).
			Scan(&b.ID, &b.Title, &b.URL, &b.CreatedAt)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				ctx.JSON(http.StatusNotFound, gin.H{"error": "Bookmark not found"})
				return
			}
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
			return
		}

		ctx.JSON(http.StatusOK, b)
	})

	// Update a bookmark by ID
	router.PUT("/bookmarks/:id", func(ctx *gin.Context) {
		id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
			return
		}

		var input Bookmark
		if err := ctx.ShouldBindJSON(&input); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var updated Bookmark
		err = db.QueryRow(ctx.Request.Context(),
			`UPDATE bookmarks SET title = $1, url = $2
			 WHERE id = $3
			 RETURNING id, title, url, created_at`,
			input.Title, input.URL, id).
			Scan(&updated.ID, &updated.Title, &updated.URL, &updated.CreatedAt)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				ctx.JSON(http.StatusNotFound, gin.H{"error": "Bookmark not found"})
				return
			}
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
			return
		}

		ctx.JSON(http.StatusOK, updated)
	})

	// Delete a bookmark by ID
	router.DELETE("/bookmarks/:id", func(ctx *gin.Context) {
		id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
			return
		}

		tag, err := db.Exec(ctx.Request.Context(),
			"DELETE FROM bookmarks WHERE id = $1", id)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
			return
		}
		if tag.RowsAffected() == 0 {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Bookmark not found"})
			return
		}

		ctx.JSON(http.StatusOK, gin.H{"message": "Bookmark deleted"})
	})

	// Create a bookmark
	router.POST("/bookmarks", func(ctx *gin.Context) {
		var input Bookmark
		if err := ctx.ShouldBindJSON(&input); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Postgres generates id and created_at; RETURNING hands them back.
		var created Bookmark
		err := db.QueryRow(ctx.Request.Context(),
			`INSERT INTO bookmarks (title, url)
			 VALUES ($1, $2)
			 RETURNING id, title, url, created_at`,
			input.Title, input.URL).
			Scan(&created.ID, &created.Title, &created.URL, &created.CreatedAt)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
			return
		}

		ctx.JSON(http.StatusCreated, created)
	})

	// Start the server
	if err := router.Run(":3000"); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
