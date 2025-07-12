package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	_ "github.com/apadilah30/ticket-booking-backend/payment-service/docs"
	"github.com/gofiber/fiber/v2"
	fiberSwagger "github.com/swaggo/fiber-swagger"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type Payment struct {
	gorm.Model
	BookingID uint    `json:"booking_id"`
	Amount    float64 `json:"amount"`
	Status    string  `json:"status"` // PENDING, SUCCESS, FAILED
}

var DB *gorm.DB

// @title Payment Service API
// @version 1.0
// @description Service for simulating payment processing.
// @host localhost:8003
// @BasePath /
func main() {
	dsn := "user:password@tcp(127.0.0.1:3306)/concert_db?charset=utf8mb4&parseTime=True&loc=Local"
	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}
	DB.AutoMigrate(&Payment{})

	app := fiber.New()
	app.Get("/swagger/*", fiberSwagger.WrapHandler)
	app.Post("/payments", CreatePayment)

	log.Println("Payment Service running on port 8003")
	app.Listen(":8003")
}

// CreatePayment godoc
// @Summary Process a payment
// @Description Simulates a payment process for a booking.
// @Tags payments
// @Accept json
// @Produce json
// @Param payment body Payment true "Payment Info"
// @Success 202 {object} Payment
// @Router /payments [post]
func CreatePayment(c *fiber.Ctx) error {
	payment := new(Payment)
	if err := c.BodyParser(payment); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot parse JSON"})
	}
	payment.Status = "PENDING"
	DB.Create(&payment)
	log.Printf("Payment received for booking ID %d. Processing...", payment.BookingID)

	// Jalankan proses di background agar bisa langsung return response
	go processPayment(payment)

	return c.Status(fiber.StatusAccepted).JSON(payment)
}

func processPayment(payment *Payment) {
	// Simulasi waktu proses pembayaran (misal: 2 detik)
	time.Sleep(2 * time.Second)

	// Simulasi hasil pembayaran (misal: selalu sukses untuk MVP)
	payment.Status = "SUCCESS"
	DB.Save(&payment)
	log.Printf("Payment for booking ID %d SUCCESS.", payment.BookingID)

	// Panggil Booking Service untuk update status
	updateBookingURL := fmt.Sprintf("http://localhost:8002/bookings/%d/status", payment.BookingID)
	payload := map[string]string{"status": "CONFIRMED"}
	jsonPayload, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPatch, updateBookingURL, bytes.NewBuffer(jsonPayload))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		log.Printf("ERROR: Failed to update booking status for ID %d. Please check manually.", payment.BookingID)
		// Implementasikan mekanisme retry atau notifikasi di sini
		return
	}
	log.Printf("Successfully updated booking status for ID %d to CONFIRMED.", payment.BookingID)
}
