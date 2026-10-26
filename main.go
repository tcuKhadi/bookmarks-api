package main

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Hello from Gin!"})
	})

	// Path parameter: /hello/John
	router.GET("/hello/:name", func(c *gin.Context) {
		// Path parameter
		name := c.Param("name")

		// Query Strings
		loud := c.DefaultQuery("loud", "false")

		greeting := "Hello, " + name + "!"
		if loud == "true" {
			greeting = strings.ToUpper(greeting)
		}

		c.JSON(http.StatusOK, gin.H{"greeting": greeting})
	})

	if err := router.Run(":3000"); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
