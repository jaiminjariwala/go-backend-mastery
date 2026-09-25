/*
	database/databaseConnection.go - this file is the power plug to MongoDB
	The problem it solves: connecting to a database is slow and expensive. You don't want every signup/login opening fresh connection. So this file connects once when the app starts, and every other file reuses that single connection.
*/

package database

import (
	"fmt"
	"log"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// DBinstance connects to MongoDB once and returns the live client(connection)
// "This function takes nothing and returns a pointer to a mongo client (*mongo.Client) - the client is just the handle for your live connection."
/*
	What does *mongo.Client mean ?
	-> the * means pointer - the address where the real client lives in memory, not a copy of it. the client is big and stateful (it manages a pool of open sockets). You want the entire app sharing one connection, so everyone passes around the address of the single client instead of copying it.
*/
func DBinstance() *mongo.Client {
	// Where your MongoDB lives. Local install = localhost:27017.
	// (MongoDB Atlas gives you a longer mongodb+srv:// address instead.)
	// Dial mongodb://localhost:27017 and hand me the connection. Go returns 2 things here: the client and a possible error - Go does this for almost every operation that can fail.
	MongoDb := "mongodb://localhost:27017"

	client, err := mongo.Connect(options.Client().ApplyURI(MongoDb))

	// if dialing failed, print the error and kill the whole app. No database = nothing works, so crashing loudly at startup is correct here.
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Connected to MongoDB!")
	return client
}

// Client is THE one shared connection for the whole app.
// Every file that touches the DB reuses this instead of reconnecting.
// this line runs once at startup and stores the connection in a package-level variable. Every file that touches the DB grabs database.Client instead of reconnecting. One plug, shared by the whole house.
var Client *mongo.Client = DBinstance()

// OpenCollection picks one collection (a "table") out of a database.
// "This function takes the client and a collection name, returns that collection."
func OpenCollection(client *mongo.Client, collectionName string) *mongo.Collection {
	collection := client.Database("cluster0").Collection(collectionName)
	return collection
}