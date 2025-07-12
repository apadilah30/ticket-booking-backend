package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	fiberSwagger "github.com/swaggo/fiber-swagger"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	_ "github.com/apadilah30/ticket-booking-backend/user-service/docs"
)

// User struct
type User struct {
	gorm.Model
	Name  string `json:"name"`
	Email string `json:"email" gorm:"unique"`
}

var DB *gorm.DB

// @title User Service API
// @version 1.0
// @description This is a user service API for managing users in concert ticket booking
// @host localhost:8001
// @BasePath /
func main() {
	dsn := "user:password@tcp(127.0.0.1:3306)/concert_db?charset=utf8mb4&parseTime=True&loc=Local"
	var err error

	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	DB.AutoMigrate(&User{})

	app := fiber.New()

	app.Get("/swagger/*", fiberSwagger.WrapHandler)

	app.Post("/users/", CreateUser)
	app.Get("/users/:id", GetUser)

	log.Println("Starting server on :8001")
	app.Listen(":8001")
}

// CreateUser godoc
// @Summary Create a new user
// @Description Create a new user in the system
// @Tags users
// @Accept json
// @Produce json
// @Param user body User true "User"
// @Success 201 {object} User
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /users/ [post]
func CreateUser(c *fiber.Ctx) error {
	user := new(User)

	if err := c.BodyParser(user); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Failed parse JSON",
		})
	}

	result := DB.Create(&user)
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": result.Error.Error(),
		})
	}

	log.Printf("User created: %v", user)

	return c.Status(fiber.StatusCreated).JSON(user)
}

// GetUser godoc
// @Summary Get user by ID
// @Description Get user details by ID
// @Tags users
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} User
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /users/{id} [post]
func GetUser(c *fiber.Ctx) error {
	id := c.Params("id")
	var user User

	result := DB.First(&user, id)

	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": result.Error.Error(),
		})
	}

	return c.JSON(user)
}
