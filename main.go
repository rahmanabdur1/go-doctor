package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Global variables for database collections (will be initialized later)
var (
	appointmentOptionsCollection *mongo.Collection
	bookingCollactions           *mongo.Collection
	usersCollactions             *mongo.Collection
	doctorsCollactions           *mongo.Collection
	paymentCollection            *mongo.Collection
	contactCollection            *mongo.Collection
	jwtSecret                    string // For JWT secret key
	mongoClient                  *mongo.Client
)

func main() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	uri := os.Getenv("DB_URI")
	if uri == "" {
		log.Fatal("DB_URI environment variable not set")
	}

	jwtSecret = os.Getenv("ACCESS_TOKEN")
	if jwtSecret == "" {
		log.Fatal("ACCESS_TOKEN environment variable not set")
	}

	// Initialize MongoDB connection
	mongoClient, err = connectMongoDB(uri)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer func() {
		if err := mongoClient.Disconnect(context.Background()); err != nil {
			log.Fatalf("Failed to disconnect from MongoDB: %v", err)
		}
	}()

	// Initialize database collections
	db := mongoClient.Database("doctors-portal")
	appointmentOptionsCollection = db.Collection("appointmentCollection")
	bookingCollactions = db.Collection("bookingCollaction")
	usersCollactions = db.Collection("usersCollaction")
	doctorsCollactions = db.Collection("doctorsCollactions")
	paymentCollection = db.Collection("paymentCollection")
	contactCollection = db.Collection("contactCollection")

	// Setup Gin router
	router := gin.Default()
	router.Use(cors.Default()) // Enable CORS

	// Define API routes (handlers are defined in handlers.go)
	setupRoutes(router)

	fmt.Printf("Doctors portal server is running on port %s\n", port)
	router.Run(":" + port)
}

// connectMongoDB establishes a connection to MongoDB
func connectMongoDB(uri string) (*mongo.Client, error) {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}
	// Check the connection
	err = client.Ping(context.Background(), nil)
	if err != nil {
		return nil, err
	}
	fmt.Println("Connected to MongoDB!")
	return client, nil
}

// setupRoutes defines all the API endpoints
func setupRoutes(router *gin.Engine) {
	router.POST("/contact", handleContactPost)
	router.GET("/appointmentOptions", handleGetAppointmentOptions)
	router.GET("/v2/appointmentOptions", handleGetV2AppointmentOptions)
	router.GET("/bookings", verifyJWT(), handleGetBookings)
	router.GET("/bookings/:id", handleGetBookingByID)
	router.POST("/bookings", handlePostBooking)
	router.POST("/create-payment-intent", handleCreatePaymentIntent)
	router.POST("/payments", handlePostPayment)
	router.GET("/jwt", handleGetJWT)
	router.GET("/appointmentSpecialty", handleGetAppointmentSpecialty)
	router.GET("/users", handleGetUsers)
	router.POST("/users", handlePostUser)
	router.GET("/users/admin/:email", handleGetUserAdminByEmail)
	router.PUT("/users/admin/:id", verifyJWT(), verifyAdmin(), handlePutUserAdminByID)
	router.GET("/doctors", verifyJWT(), verifyAdmin(), handleGetDoctors)
	router.DELETE("/doctors/:id", verifyJWT(), verifyAdmin(), handleDeleteDoctorByID)
	router.POST("/doctors", verifyJWT(), verifyAdmin(), handlePostDoctor)
}
