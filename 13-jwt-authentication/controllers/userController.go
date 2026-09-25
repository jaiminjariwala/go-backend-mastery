package controllers

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/jaiminjariwala/go-backend-mastery/13-jwt-authentication/database"
	"github.com/jaiminjariwala/go-backend-mastery/13-jwt-authentication/helpers"
	"github.com/jaiminjariwala/go-backend-mastery/13-jwt-authentication/models"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"golang.org/x/crypto/bcrypt"
)

// userCollection is this file's handle on the "user" collection.
var userCollection *mongo.Collection = database.OpenCollection(database.Client, "user")

// validate checks the `validate:"..."` struct tags (required, min, email...).
var validate = validator.New()

// HashPassword turns a plain password into a bcrypt hash.
// "This function takes a plain password and returns its hash."
// Cost 14 = how much work hashing takes (higher = slower = safer).
func HashPassword(password string) string {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		log.Panic(err)
	}
	return string(bytes)
}

// VerifyPassword checks a login attempt against the stored hash.
// "This function takes the plain password and the stored hash,
//  returns (true, '') when they match."
func VerifyPassword(userPassword string, providedPassword string) (bool, string) {
	err := bcrypt.CompareHashAndPassword([]byte(providedPassword), []byte(userPassword))
	if err != nil {
		return false, "login or password is incorrect"
	}
	return true, ""
}

// Signup handles POST /users/signup.
// "This function takes nothing and returns a gin.HandlerFunc."
func Signup() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Every DB call gets a 100-second deadline; cancel it when we return.
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel() // when this handler returns, cancel the DB timer

		var user models.User

		// Decode the request JSON body into our User struct
		// (using the `json:"..."` tags). Bad JSON = 400.
		if err := c.ShouldBindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Run the `validate:"..."` tag rules: required, min/max length, email...
		if err := validate.Struct(user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Reject the signup if this email is already registered.
		count, err := userCollection.CountDocuments(ctx, bson.M{"email": user.Email})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occurred while checking for the email"})
			return
		}
		if count > 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "this email already exists"})
			return
		}

		// Reject the signup if this phone number is already registered.
		count, err = userCollection.CountDocuments(ctx, bson.M{"phone": user.Phone})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occurred while checking for the phone number"})
			return
		}
		if count > 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "this phone number already exists"})
			return
		}

		// Hash the password BEFORE storing. Plain passwords never touch the DB.
		password := HashPassword(*user.Password)
		user.Password = &password

		// Fill in the server-generated fields: timestamps and ids.
		user.Created_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		user.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		user.ID = bson.NewObjectID() // v2: the video calls primitive.NewObjectID()
		user.User_id = user.ID.Hex() // the hex-string form, used in URLs and filters

		// Mint the JWTs and store them on the user document.
		token, refreshToken, _ := helpers.GenerateAllTokens(*user.Email, *user.First_name, *user.Last_name, *user.User_type, user.User_id)
		user.Token = &token
		user.Refresh_token = &refreshToken

		// Insert the finished document into MongoDB.
		resultInsertionNumber, err := userCollection.InsertOne(ctx, user)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "user was not created"})
			return
		}

		c.JSON(http.StatusOK, resultInsertionNumber)
	}
}

// Login handles POST /users/login.
// The body carries {"email": ..., "password": ...} (bound into a User).
// "This function takes nothing and returns a gin.HandlerFunc."
func Login() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel() // when this handler returns, cancel the DB timer

		var user models.User
		var foundUser models.User

		if err := c.ShouldBindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Find the registered user with this email. One document expected.
		err := userCollection.FindOne(ctx, bson.M{"email": user.Email}).Decode(&foundUser)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "login or password is incorrect"})
			return
		}

		// Compare the given password against the stored bcrypt hash.
		passwordIsValid, msg := VerifyPassword(*user.Password, *foundUser.Password)
		if !passwordIsValid {
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			return
		}

		// Password ok: mint fresh tokens and save them onto the user.
		token, refreshToken, _ := helpers.GenerateAllTokens(*foundUser.Email, *foundUser.First_name, *foundUser.Last_name, *foundUser.User_type, foundUser.User_id)
		helpers.UpdateAllTokens(token, refreshToken, foundUser.User_id)

		c.JSON(http.StatusOK, foundUser)
	}
}

// GetUsers handles GET /users (ADMIN only).
// Pagination via query params: ?page=1&recordPerPage=10&startIndex=0
// "This function takes nothing and returns a gin.HandlerFunc."
func GetUsers() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only admins may list every user.
		if err := helpers.CheckUserType(c, "ADMIN"); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel() // when this handler returns, cancel the DB timer

		// Read pagination params, with sane defaults.
		recordPerPage, err := strconv.Atoi(c.Query("recordPerPage"))
		if err != nil || recordPerPage < 1 {
			recordPerPage = 10
		}
		page, err := strconv.Atoi(c.Query("page"))
		if err != nil || page < 1 {
			page = 1
		}
		startIndex := (page - 1) * recordPerPage // which record the page starts at
		if s := c.Query("startIndex"); s != "" {
			startIndex, err = strconv.Atoi(s)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "startIndex must be a number"})
				return
			}
		}

		// Aggregation pipeline: three stages that run inside MongoDB.
		// 1. $match:   keep every document (empty filter).
		// 2. $group:   squash them into one doc with total_count + all rows in "data".
		// 3. $project: reshape: drop _id, keep total_count, slice "data" into one page.
		matchStage := bson.D{{Key: "$match", Value: bson.D{}}}
		groupStage := bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: bson.D{{Key: "_id", Value: "null"}}},
			{Key: "total_count", Value: bson.D{{Key: "$sum", Value: 1}}},
			{Key: "data", Value: bson.D{{Key: "$push", Value: "$$ROOT"}}},
		}}}
		projectStage := bson.D{{Key: "$project", Value: bson.D{
			{Key: "_id", Value: 0},
			{Key: "total_count", Value: 1},
			{Key: "user_items", Value: bson.D{{Key: "$slice", Value: []interface{}{"$data", startIndex, recordPerPage}}}},
		}}}

		result, err := userCollection.Aggregate(ctx, mongo.Pipeline{matchStage, groupStage, projectStage})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occurred while listing user items"})
			return
		}

		// All decodes every remaining document from the cursor at once.
		var allUsers []bson.M
		if err := result.All(ctx, &allUsers); err != nil {
			log.Fatal(err)
		}

		c.JSON(http.StatusOK, allUsers[0])
	}
}

// GetUser handles GET /users/:user_id.
// A USER may only fetch themselves; an ADMIN may fetch anyone.
// "This function takes nothing and returns a gin.HandlerFunc."
func GetUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := c.Param("user_id") // the :user_id part of the URL

		// Forbidden when a USER asks for somebody else's record.
		if err := helpers.MatchUserTypeToUid(c, userId); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel() // when this handler returns, cancel the DB timer

		var user models.User

		// Find the one document with this user_id and decode it into user.
		err := userCollection.FindOne(ctx, bson.M{"user_id": userId}).Decode(&user)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, user)
	}
}
