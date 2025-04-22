package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func handleContactPost(c *gin.Context) {
	var contact Contact
	if err := c.BindJSON(&contact); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := contactCollection.InsertOne(context.Background(), contact)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to insert contact"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func handleGetAppointmentOptions(c *gin.Context) {
	date := c.Query("date")
	if date == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "date query parameter is required"})
		return
	}

	var options []AppointmentOption
	cursor, err := appointmentOptionsCollection.Find(context.Background(), bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch appointment options"})
		return
	}
	if err = cursor.All(context.Background(), &options); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to decode appointment options"})
		return
	}
	defer cursor.Close(context.Background())

	var alreadyBooked []Booking
	bookingQuery := bson.M{"appointmentDate": date}
	bookingCursor, err := bookingCollactions.Find(context.Background(), bookingQuery)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch bookings"})
		return
	}
	if err = bookingCursor.All(context.Background(), &alreadyBooked); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to decode bookings"})
		return
	}
	defer bookingCursor.Close(context.Background())

	for i := range options {
		var bookedSlots []string
		for _, book := range alreadyBooked {
			if book.Treatment == options[i].Name {
				bookedSlots = append(bookedSlots, book.Slot)
			}
		}
		var remainingSlots []string
		for _, slot := range options[i].Slots {
			found := false
			for _, bookedSlot := range bookedSlots {
				if slot == bookedSlot {
					found = true
					break
				}
			}
			if !found {
				remainingSlots = append(remainingSlots, slot)
			}
		}
		options[i].Slots = remainingSlots
		fmt.Println(date, options[i].Name, len(remainingSlots))
	}

	c.JSON(http.StatusOK, options)
}

func handleGetV2AppointmentOptions(c *gin.Context) {
	date := c.Query("data")
	if date == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "data query parameter is required"})
		return
	}

	pipeline := []bson.M{
		{"$lookup": bson.M{
			"from":         "bookingCollaction",
			"localField":   "name",
			"foreignField": "treatment",
			"pipeline": []bson.M{
				{"$match": bson.M{
					"$expr": bson.M{
						"$eq": []interface{}{"$appointmentDate", date},
					},
				}},
			},
			"as": "booked",
		}},
		{"$project": bson.M{
			"name":  1,
			"slots": 1,
			"price": 1,
			"booked": bson.M{
				"$map": bson.M{
					"input": "<span class=\"math-inline\">booked",
					"as":    "book",
					"in":    "</span>$book.slot",
				},
			},
		}},
		{"$project": bson.M{
			"name":  1,
			"price": 1,
			"slots": bson.M{
				"$setDifference": []interface{}{"$slots", "$booked"},
			},
		}},
	}

	cursor, err := appointmentOptionsCollection.Aggregate(context.Background(), pipeline)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to aggregate appointment options"})
		return
	}
	defer cursor.Close(context.Background())

	var options []AppointmentOption
	if err = cursor.All(context.Background(), &options); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to decode aggregated options"})
		return
	}

	c.JSON(http.StatusOK, options)
}

func handleGetBookings(c *gin.Context) {
	email := c.Query("email")
	decodedEmail, exists := c.Get("decodedEmail")
	if !exists {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized access"})
		return
	}

	if email != decodedEmail {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	var bookings []Booking
	cursor, err := bookingCollactions.Find(context.Background(), bson.M{"email": email})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch bookings"})
		return
	}
	if err = cursor.All(context.Background(), &bookings); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to decode bookings"})
		return
	}
	defer cursor.Close(context.Background())

	c.JSON(http.StatusOK, bookings)
}

func handleGetBookingByID(c *gin.Context) {
	idStr := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid booking ID"})
		return
	}

	var booking Booking
	err = bookingCollactions.FindOne(context.Background(), bson.M{"_id": objID}).Decode(&booking)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "booking not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch booking"})
		}
		return
	}

	c.JSON(http.StatusOK, booking)
}

func handlePostBooking(c *gin.Context) {
	var booking Booking
	if err := c.BindJSON(&booking); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := bson.M{
		"appointmentDate": booking.AppointmentDate,
		"email":           booking.Email,
		"treatment":       booking.Treatment,
	}

	var existingBooking Booking
	err := bookingCollactions.FindOne(context.Background(), query).Decode(&existingBooking)
	if err == nil {
		message := fmt.Sprintf("You already have a booking on %s", booking.AppointmentDate)
		c.JSON(http.StatusOK, gin.H{"acknowledged": false, "message": message})
		return
	} else if err != mongo.ErrNoDocuments {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check existing booking"})
		return
	}

	result, err := bookingCollactions.InsertOne(context.Background(), booking)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to insert booking"})
		return
	}

	// TODO: Implement sendBookingEmail function (requires external email service integration)
	// sendBookingEmail(booking)

	c.JSON(http.StatusOK, result)
}

func handleCreatePaymentIntent(c *gin.Context) {
	var booking Booking
	if err := c.BindJSON(&booking); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	price := booking.Price
	amount := int64(price * 100)

	// TODO: Integrate with Stripe to create a payment intent
	// You'll need to install the Stripe Go library and use your secret key
	// Example (replace with your actual Stripe API call):
	// stripe.Key = os.Getenv("STRIPE_KEY")
	// params := &stripe.PaymentIntentParams{
	// 	Amount:   stripe.Int64(amount),
	// 	Currency: stripe.String(stripe.CurrencyUSD),
	// 	PaymentMethodTypes: stripe.StringSlice([]string{
	// 		"card",
	// 	}),
	// }
	// pi, err := paymentintent.New(params)
	// if err != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create payment intent"})
	// 	log.Printf("Stripe error: %v", err)
	// 	return
	// }
	//
	// c.JSON(http.StatusOK, gin.H{"clientSecret": pi.ClientSecret})

	// Placeholder for demonstration without actual Stripe integration
	fmt.Printf("Creating payment intent for amount: %d\n", amount)
	clientSecret := "test_client_secret" // Placeholder
	c.JSON(http.StatusOK, gin.H{"clientSecret": clientSecret})
}

func handlePostPayment(c *gin.Context) {
	var payment Payment
	if err := c.BindJSON(&payment); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := paymentCollection.InsertOne(context.Background(), payment)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to insert payment"})
		return
	}

	// TODO: Update booking status to paid if needed

	c.JSON(http.StatusOK, result)
}

func handleGetJWT(c *gin.Context) {
	email := c.Query("email")
	if email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email query parameter is required"})
		return
	}

	var user User
	err := usersCollactions.FindOne(context.Background(), bson.M{"email": email}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusUnauthorized, gin.H{"accessToken": ""})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to find user"})
		}
		return
	}

	expirationTime := time.Now().Add(2 * 24 * time.Hour) // 2 days
	claims := &Claims{
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate JWT"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"accessToken": tokenString})
}

func handleGetAppointmentSpecialty(c *gin.Context) {
	cursor, err := appointmentOptionsCollection.Find(context.Background(), bson.M{}, options.Find().SetProjection(bson.M{"name": 1, "_id": 0}))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch appointment specialties"})
		return
	}
	defer cursor.Close(context.Background())

	var specialties []map[string]string
	for cursor.Next(context.Background()) {
		var option AppointmentOption
		if err := cursor.Decode(&option); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to decode specialty"})
			return
		}
		specialties = append(specialties, map[string]string{"name": option.Name})
	}

	if err := cursor.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cursor error"})
		return
	}

	c.JSON(http.StatusOK, specialties)
}

func handleGetUsers(c *gin.Context) {
	cursor, err := usersCollactions.Find(context.Background(), bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch users"})
		return
	}
	defer cursor.Close(context.Background())

	var users []User
	if err = cursor.All(context.Background(), &users); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to decode users"})
		return
	}

	c.JSON(http.StatusOK, users)
}

func handlePostUser(c *gin.Context) {
	var user User
	if err := c.BindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := usersCollactions.InsertOne(context.Background(), user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to insert user"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func handleGetUserAdminByEmail(c *gin.Context) {
	email := c.Param("email")
	var user User
	err := usersCollactions.FindOne(context.Background(), bson.M{"email": email}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusOK, gin.H{"isAdmin": false})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to find user"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"isAdmin": user.Role == "admin"})
}

func handlePutUserAdminByID(c *gin.Context) {
	idStr := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	filter := bson.M{"_id": objID}
	update := bson.M{"$set": bson.M{"role": "admin"}}
	options := options.Update().SetUpsert(true)

	result, err := usersCollactions.UpdateOne(context.Background(), filter, update, options)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user role"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func handleGetDoctors(c *gin.Context) {
	cursor, err := doctorsCollactions.Find(context.Background(), bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch doctors"})
		return
	}
	defer cursor.Close(context.Background())

	var doctors []Doctor
	if err = cursor.All(context.Background(), &doctors); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to decode doctors"})
		return
	}

	c.JSON(http.StatusOK, doctors)
}

func handleDeleteDoctorByID(c *gin.Context) {
	idStr := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid doctor ID"})
		return
	}

	filter := bson.M{"_id": objID}
	result, err := doctorsCollactions.DeleteOne(context.Background(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete doctor"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func handlePostDoctor(c *gin.Context) {
	var doctor Doctor
	if err := c.BindJSON(&doctor); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := doctorsCollactions.InsertOne(context.Background(), doctor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to insert doctor"})
		return
	}

	c.JSON(http.StatusOK, result)
}
