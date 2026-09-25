package helpers

import (
	"errors"

	"github.com/gin-gonic/gin"
)

// CheckUserType makes sure the caller has the required role.
// "This function takes the request context and a role,
//  returns an error when the caller's user_type does not match."
// A nil error means: allowed.
func CheckUserType(c *gin.Context, role string) (err error) {
	// user_type was stashed into the context by the auth middleware.
	userType := c.GetString("user_type")
	if userType != role {
		err = errors.New("unauthorized to access this resource")
	}
	return err
}

// MatchUserTypeToUid makes sure a normal USER can only fetch their own record.
// "This function takes the context and the requested user_id,
//  returns an error when a USER asks for someone else's record."
func MatchUserTypeToUid(c *gin.Context, userId string) (err error) {
	userType := c.GetString("user_type")
	uid := c.GetString("uid") // the caller's own id, from their token

	// A USER asking for somebody else's record: forbidden.
	if userType == "USER" && uid != userId {
		err = errors.New("unauthorized to access this resource")
		return err
	}

	// Otherwise run the normal role check (keeps ADMIN-only routes safe).
	err = CheckUserType(c, userType)
	return err
}
