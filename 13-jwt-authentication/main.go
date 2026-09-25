package main

import (
	"os"

	"github.com/gin-gonic/gin"

	"github.com/jaiminjariwala/go-backend-mastery/13-jwt-authentication/routes"
)

func main() {
	// Read PORT from the environment (hosting sites set this for you).
	// Empty on your laptop, so fall back to 8000.
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	router := gin.New()      // create the router: it maps URLs to handler functions
	router.Use(gin.Logger()) // middleware: log every request to the terminal

	routes.AuthRoutes(router) // public doors: signup and login (no token needed)
	routes.UserRoutes(router) // locked doors: every route here needs a valid JWT

	// Demo routes from the video: they exist only to test the middleware.
	// They sit AFTER UserRoutes, so they are protected too: no token = blocked.
	router.GET("/api-1", func(c *gin.Context) {
		c.JSON(200, gin.H{"success": "Access granted for api-1"})
	})

	router.GET("/api-2", func(c *gin.Context) {
		c.JSON(200, gin.H{"success": "Access granted for api-2"})
	})

	// Start serving on :8000. This line blocks forever, like http.ListenAndServe.
	router.Run(":" + port)
}
