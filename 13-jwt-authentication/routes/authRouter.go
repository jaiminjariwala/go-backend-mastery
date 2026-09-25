package routes

import (
	"github.com/gin-gonic/gin"

	"github.com/jaiminjariwala/go-backend-mastery/13-jwt-authentication/controllers"
)

// AuthRoutes wires the public endpoints: signup and login need no token.
func AuthRoutes(incomingRoutes *gin.Engine) {
	incomingRoutes.POST("/users/signup", controllers.Signup())
	incomingRoutes.POST("/users/login", controllers.Login())
}
