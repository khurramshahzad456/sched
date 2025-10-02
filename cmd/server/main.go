package main

import (
	"context"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"scheduler-service/internal/app"
	"scheduler-service/internal/server"	
)

func main() {
	ctx := context.Background()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL required")
	}

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}
	defer pool.Close()

	appInstance := &app.App{DB: pool}

	router := gin.Default()
	
	// OAuth2 callback (must be before auth middleware)
	router.GET("/oauth2callback", appInstance.GoogleOAuth2CallbackHandler)
	
	router.Use(app.AuthMiddlewareFromEnv())

	api := router.Group("/api")
	{
		users := api.Group("/users")
		{
			users.POST("/:id/availability", appInstance.SetAvailabilityHandler)
			users.PUT("/:id/availability/:rule_id", appInstance.UpdateAvailabilityHandler)
			users.GET("/:id/availability", appInstance.ListAvailabilityHandler)
			users.GET("/:id/slots", appInstance.GetSlotsHandler)
			users.POST("/:id/bookings", appInstance.CreateBookingHandler)
			users.GET("/:id/bookings", appInstance.ListBookingsHandler)
		}
		api.DELETE("/bookings/:id", appInstance.CancelBookingHandler)
		
		// Google Calendar integration routes
		calendar := api.Group("/calendar")
		{
			calendar.GET("/auth", appInstance.GoogleAuthHandler)
			calendar.GET("/events", appInstance.GetGoogleCalendarEvents)
			calendar.GET("/calendars", appInstance.GetGoogleCalendarList)
		}
	}

	server.Run(router)
}
