package main

import "go.mongodb.org/mongo-driver/bson/primitive"

// AppointmentOption represents the structure of appointment options
type AppointmentOption struct {
	ID    primitive.ObjectID `bson:"_id,omitempty"`
	Name  string             `bson:"name"`
	Slots []string           `bson:"slots"`
	Price float64            `bson:"price"`
}

// Booking represents the structure of a booking
type Booking struct {
	ID              primitive.ObjectID `bson:"_id,omitempty"`
	AppointmentDate string             `bson:"appointmentDate"`
	Treatment       string             `bson:"treatment"`
	Patient         string             `bson:"patient"`
	Slot            string             `bson:"slot"`
	Email           string             `bson:"email"`
	Phone           string             `bson:"phone"`
	Price           float64            `bson:"price"`
}

// User represents the structure of a user
type User struct {
	ID    primitive.ObjectID `bson:"_id,omitempty"`
	Name  string             `bson:"name"`
	Email string             `bson:"email"`
	Role  string             `bson:"role"`
}

// Doctor represents the structure of a doctor
type Doctor struct {
	ID    primitive.ObjectID `bson:"_id,omitempty"`
	Name  string             `bson:"name"`
	Email string             `bson:"email"`
	Image string             `bson:"img"`
}

// Contact represents the structure of a contact message
type Contact struct {
	ID      primitive.ObjectID `bson:"_id,omitempty"`
	Name    string             `bson:"name"`
	Email   string             `bson:"email"`
	Subject string             `bson:"subject"`
	Message string             `bson:"message"`
}

// Payment represents the structure of a payment record
type Payment struct {
	ID              primitive.ObjectID `bson:"_id,omitempty"`
	PaymentMethodId string             `bson:"paymentMethodId"`
	Booking         PaymentBooking     `bson:"booking"`
}

// PaymentBooking represents the embedded booking information in the Payment model
type PaymentBooking struct {
	ID              string  `bson:"_id"`
	AppointmentDate string  `bson:"appointmentDate"`
	Treatment       string  `bson:"treatment"`
	Patient         string  `bson:"patient"`
	Slot            string  `bson:"slot"`
	Email           string  `bson:"email"`
	Phone           string  `bson:"phone"`
	Price           float64 `bson:"price"`
}
