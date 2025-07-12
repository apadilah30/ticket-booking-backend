package main

import (
	"log"
	"net/http"
	"bytes"
	"encoding/json"
  "time"

	"github.com/gofiber/fiber/v2"
	fiberSwagger "github.com/swaggo/fiber-swagger"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	_ "github.com/apadilah30/ticket-booking-backend/booking-service/docs"
)

type Booking struct {
	gorm.Model
	UserID       uint   `json:"user_id"`
	ConcertName  string `json:"concert_name"`
	TicketCount  int    `json:"ticket_count"`
	Status       string `json:"status"` // PENDING, CONFIRMED, CANCELLED
	TotalPrice   float64 `json:"total_price"`
}

type PaymentRequest struct {
    BookingID uint `json:"booking_id"`
    Amount float64 `json:"amount"`
}

var DB *gorm.DB

// @title Booking Service API
// @version 1.0
// @description Service for managing concert ticket bookings.
// @host localhost:8002
// @BasePath /
func main() {
	dsn := "user:password@tcp(127.0.0.1:3306)/concert_db?charset=utf8mb4&parseTime=True&loc=Local"
	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}
	DB.AutoMigrate(&Booking{})

	app := fiber.New()
	app.Get("/swagger/*", fiberSwagger.WrapHandler)
	app.Post("/bookings", CreateBooking)
	app.Get("/bookings/:id", GetBooking)
	app.Patch("/bookings/:id/status", UpdateBookingStatus) // Endpoint internal

	log.Println("Booking Service running on port 8002")
	app.Listen(":8002")
}

// CreateBooking godoc
// @Summary Create a new booking
// @Description Creates a new booking with PENDING status and triggers payment.
// @Tags bookings
// @Accept json
// @Produce json
// @Param booking body Booking true "Booking Info"
// @Success 201 {object} Booking
// @Router /bookings [post]
func CreateBooking(c *fiber.Ctx) error {
    booking := new(Booking)
    if err := c.BodyParser(booking); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot parse JSON"})
    }

    booking.Status = "PENDING"
    // Asumsi harga tiket Rp 500.000
    const ticketPrice = 500000
    booking.TotalPrice = float64(booking.TicketCount) * ticketPrice

    if err := DB.Create(&booking).Error; err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not create booking"})
    }
    log.Printf("Booking created with ID %d, status PENDING", booking.ID)

    // Asynchronous call to payment service
    go triggerPayment(booking.ID, booking.TotalPrice)

    return c.Status(fiber.StatusCreated).JSON(booking)
}

func triggerPayment(bookingID uint, amount float64) {
    time.Sleep(1 * time.Second) // Small delay to ensure booking is committed
    log.Printf("Triggering payment for booking ID %d for amount %.2f", bookingID, amount)
    paymentURL := "http://localhost:8003/payments"

    payload := PaymentRequest{
        BookingID: bookingID,
        Amount:    amount,
    }
    jsonPayload, _ := json.Marshal(payload)

    _, err := http.Post(paymentURL, "application/json", bytes.NewBuffer(jsonPayload))
    if err != nil {
        log.Printf("Failed to trigger payment for booking ID %d: %v", bookingID, err)
        // Lakukan retry logic atau tandai booking sebagai FAILED
    }
}

// GetBooking godoc
// @Summary Get a booking by ID
// @Description Get booking details by ID
// @Tags bookings
// @Produce json
// @Param id path int true "Booking ID"
// @Success 200 {object} Booking
// @Router /bookings/{id} [get]
func GetBooking(c *fiber.Ctx) error {
    id := c.Params("id")
    var booking Booking
    if err := DB.First(&booking, id).Error; err != nil {
        return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "booking not found"})
    }
    return c.JSON(booking)
}


// UpdateBookingStatus godoc
// @Summary Update booking status (internal)
// @Description Updates the status of a booking, usually called by payment-service
// @Tags bookings
// @Accept json
// @Produce json
// @Param id path int true "Booking ID"
// @Param status body map[string]string true "New Status"
// @Success 200 {object} Booking
// @Router /bookings/{id}/status [patch]
func UpdateBookingStatus(c *fiber.Ctx) error {
    id := c.Params("id")
    var booking Booking
    if err := DB.First(&booking, id).Error; err != nil {
        return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "booking not found"})
    }

    var payload struct {
        Status string `json:"status"`
    }
    if err := c.BodyParser(&payload); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid payload"})
    }

    booking.Status = payload.Status
    DB.Save(&booking)
    log.Printf("Booking ID %s status updated to %s", id, payload.Status)
    return c.JSON(booking)
}