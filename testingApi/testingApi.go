package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"gofr.dev/pkg/gofr"
)

type Customer struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

// Function to generate dummy customer data
func generateDummyCustomers(count int) []Customer {
	gofakeit.Seed(time.Now().UnixNano()) // Seed for random generation
	customers := make([]Customer, count)
	rand.Seed(time.Now().UnixNano()) // Seed with the current time
	randomID := rand.Intn(1000000)

	for i := 0; i < count; i++ {
		customers[i] = Customer{
			ID:    randomID,
			Name:  gofakeit.Name(),
			Email: gofakeit.Email(),
			Phone: gofakeit.Phone(),
		}
	}

	return customers
}

func main() {
	// Create a new GoFr app
	app := gofr.New()

	// Define an API endpoint that returns dummy customer data
	app.GET("/customers", func(c *gofr.Context) (interface{}, error) {
		// Get the number of customers to generate (default 10)
		count := 10
		// Generate dummy customer data
		customers := generateDummyCustomers(count)
		fmt.Println("customers = ", customers)
		return customers, nil
	})

	app.POST("/UploadUser", func(c *gofr.Context) (interface{}, error) {
		// Post call for upload user
		return "SUCCESSFUL", nil
	})

	// Start the GoFr server
	app.Run()
}
