package middleware

import (
	"fmt"
	"net/http"

	"github.com/jaiminjariwala/go-backend-mastery/13-jwt-authentication/helpers"

	"github.com/gin-gonic/gin"
)

// Authenticate is middleware: it runs BEFORE every protected handler.
// It reads the "token" header, validates the JWT, and stashes the user's
// identity into the request context for the handler to use.
// "This function takes nothing and returns a gin.HandlerFunc (middleware)."
func Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		// The client sends its JWT in a header literally called "token".
		clientToken := c.Request.Header.Get("token")
		if clientToken == "" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("No Authorization header provided")})
			c.Abort() // stop here: the real handler never runs
			return
		}

		claims, errMsg := helpers.ValidateToken(clientToken)
		if errMsg != "" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": errMsg})
			c.Abort()
			return
		}

		// Token is valid: stash who this is so handlers can read it with c.GetString.
		c.Set("email", claims.Email)
		c.Set("first_name", claims.First_name)
		c.Set("last_name", claims.Last_name)
		c.Set("uid", claims.Uid)
		c.Set("user_type", claims.User_type)

		c.Next() // token ok: continue to the real handler
	}
}
