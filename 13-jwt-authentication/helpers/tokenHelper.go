package helpers

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jaiminjariwala/go-backend-mastery/13-jwt-authentication/database"

	jwt "github.com/dgrijalva/jwt-go"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// SignedDetails is the luggage packed INSIDE every JWT we issue.
// It embeds jwt.StandardClaims, which adds the standard fields:
// ExpiresAt (when it dies), IssuedAt, Issuer.
type SignedDetails struct {
	Email      string
	First_name string
	Last_name  string
	Uid        string
	User_type  string
	jwt.StandardClaims
}

// userCollection is this file's handle on the "user" collection.
var userCollection *mongo.Collection = database.OpenCollection(database.Client, "user")

// SECRET_KEY signs every token. It comes from an environment variable so the
// secret never sits in your code. Set it before running:
//   export SECRET_KEY="some-long-random-string"
var SECRET_KEY string = os.Getenv("SECRET_KEY")

// GenerateAllTokens mints two JWTs: a short-lived access token (24h) and a
// long-lived refresh token (7 days).
// "This function takes the user's details and returns (token, refreshToken, err)."
func GenerateAllTokens(email string, firstName string, lastName string, userType string, uid string) (signedToken string, signedRefreshToken string, err error) {
	// The access token carries who the user is and dies in 24 hours.
	claims := &SignedDetails{
		Email:      email,
		First_name: firstName,
		Last_name:  lastName,
		Uid:        uid,
		User_type:  userType,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Local().Add(time.Hour * time.Duration(24)).Unix(),
		},
	}

	// The refresh token carries no identity, just a 7-day expiry.
	// It exists only to mint new access tokens without another login.
	refreshClaims := &SignedDetails{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Local().Add(time.Hour * time.Duration(168)).Unix(),
		},
	}

	// NewWithClaims picks the signing method (HS256 = HMAC with SHA-256).
	// SignedString signs the claims with SECRET_KEY and returns the token text.
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(SECRET_KEY))
	if err != nil {
		log.Panic(err)
		return
	}

	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString([]byte(SECRET_KEY))
	if err != nil {
		log.Panic(err)
		return
	}

	return token, refreshToken, err
}

// ValidateToken checks a token's signature and expiry, then unpacks it.
// "This function takes a token string and returns (the claims inside, a message)."
// An empty message means the token is valid.
func ValidateToken(signedToken string) (claims *SignedDetails, msg string) {
	// ParseWithClaims verifies the signature using SECRET_KEY and
	// decodes the luggage into a SignedDetails struct.
	token, err := jwt.ParseWithClaims(
		signedToken,
		&SignedDetails{},
		func(token *jwt.Token) (interface{}, error) {
			return []byte(SECRET_KEY), nil
		},
	)
	if err != nil {
		msg = err.Error()
		return
	}

	// Type-assert: the decoded claims really are our SignedDetails.
	claims, ok := token.Claims.(*SignedDetails)
	if !ok {
		msg = fmt.Sprintf("the token is invalid")
		return
	}

	// ExpiresAt is a Unix timestamp: reject tokens whose time has passed.
	if claims.ExpiresAt < time.Now().Local().Unix() {
		msg = fmt.Sprintf("token is expired")
		return
	}

	return claims, msg
}

// UpdateAllTokens writes the fresh tokens into the user's DB document.
// "This function takes both tokens and a user_id, returns nothing."
func UpdateAllTokens(signedToken string, signedRefreshToken string, userId string) {
	// Every DB call gets a 100-second deadline; cancel it when we return.
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel() // when this function returns, cancel the DB timer

	// Build the update list: $set means "change only these fields".
	var updateObj bson.D
	updateObj = append(updateObj, bson.E{Key: "token", Value: signedToken})
	updateObj = append(updateObj, bson.E{Key: "refresh_token", Value: signedRefreshToken})

	updated_at, _ := time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
	updateObj = append(updateObj, bson.E{Key: "updated_at", Value: updated_at})

	// v2 change: the video builds options.UpdateOptions{Upsert: &upsert}.
	// v2 uses a builder instead: options.UpdateOne().SetUpsert(true).
	// Upsert = "update it, or insert it if it is missing".
	opt := options.UpdateOne().SetUpsert(true)
	filter := bson.M{"user_id": userId}

	_, err := userCollection.UpdateOne(ctx, filter, bson.D{{Key: "$set", Value: updateObj}}, opt)
	if err != nil {
		log.Panic(err)
		return
	}
}
