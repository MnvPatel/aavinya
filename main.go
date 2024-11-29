package main

import (
	"log"
	"net/http"

	"github.com/adityjoshi/aavinya/consumer"
	"github.com/adityjoshi/aavinya/controllers"
	"github.com/adityjoshi/aavinya/database"
	"github.com/adityjoshi/aavinya/initiliazers"
	kafkamanager "github.com/adityjoshi/aavinya/kafka/kafkaManager"
	"github.com/adityjoshi/aavinya/routes"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

var km *kafkamanager.KafkaManager

func init() {
	initiliazers.LoadEnvVariable()
}

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	database.InitDatabase()
	defer database.CloseDatabase()
	database.InitializeRedisClient()

	northBrokers := []string{"localhost:9092"}
	southBrokers := []string{"localhost:9092"}
	var err error
	km, err = kafkamanager.NewKafkaManager(northBrokers, southBrokers)
	if err != nil {
		log.Fatal("Failed to initialize Kafka Manager:", err)
	}

	regions := []string{"north", "south"}
	for _, region := range regions {
		go func(r string) {
			log.Printf("Starting Kafka consumer for region: %s\n", r)
			consumer.StartConsumer(r)
		}(region)
	}

	go controllers.SubscribeToPaymentUpdates()
	go controllers.SubscribeToHospitalizationUpdates()
	go controllers.SubscribeToHospitaliztionUpdates()
	go controllers.SubscribeToAppointmentUpdates()
	go controllers.CheckAppointmentsQueue()
	go controllers.SubscribeToAppointmentUpdates()

	// Setup the HTTP server with Gin
	router := gin.Default()
	router.Use(setupCORS())
	setupSessions(router)
	setupRoutes(router)

	// Start server
	server := &http.Server{
		Addr:    ":2426",
		Handler: router,
	}
	log.Println("Server is running at :2426...")

	// Keep main function running indefinitely
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
	select {}
}

func setupCORS() gin.HandlerFunc {
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{
		"http://localhost:5173",
	}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"} // Allow OPTIONS
	config.AllowHeaders = append(config.AllowHeaders, "Authorization", "Content-Type", "credentials", "region")
	config.AllowCredentials = true
	return cors.New(config)
}

// setupSessions configures session management
func setupSessions(router *gin.Engine) {
	store := cookie.NewStore([]byte("secret"))
	router.Use(sessions.Sessions("session", store))
}

// Add a route for handling OPTIONS requests globall

// setupRoutes defines all application routes
func setupRoutes(router *gin.Engine) {
	routes.UserRoutes(router)
	routes.UserInfoRoutes(router)
	routes.HospitalAdmin(router, km)
}
