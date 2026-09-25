package routes

import (
	"github.com/gin-gonic/gin"

	"github.com/jaiminjariwala/go-backend-mastery/13-jwt-authentication/controllers"
	"github.com/jaiminjariwala/go-backend-mastery/13-jwt-authentication/middleware"
)

// UserRoutes wires the protected endpoints.
// Use runs middleware.Authenticate() first on every route below,
// so each one requires a valid JWT in the "token" header.
func UserRoutes(incomingRoutes *gin.Engine) {
	incomingRoutes.Use(middleware.Authenticate())
	incomingRoutes.GET("/users", controllers.GetUsers())
	incomingRoutes.GET("/users/:user_id", controllers.GetUser())
}
