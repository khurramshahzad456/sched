package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

// GoogleCalendarConfig holds OAuth2 configuration
type GoogleCalendarConfig struct {
	Config *oauth2.Config
}

// CalendarEvent represents a Google Calendar event
type CalendarEvent struct {
	ID          string    `json:"id"`
	Summary     string    `json:"summary"`
	Description string    `json:"description,omitempty"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	Location    string    `json:"location,omitempty"`
	Status      string    `json:"status"`
	Creator     string    `json:"creator,omitempty"`
}

// InitGoogleCalendarConfig initializes OAuth2 config for Google Calendar
func InitGoogleCalendarConfig() *GoogleCalendarConfig {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	redirectURL := os.Getenv("GOOGLE_REDIRECT_URL")

	if clientID == "" || clientSecret == "" || redirectURL == "" {
		return nil
	}

	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes: []string{
			calendar.CalendarReadonlyScope,
		},
		Endpoint: google.Endpoint,
	}

	return &GoogleCalendarConfig{Config: config}
}

// GoogleAuthHandler initiates OAuth2 flow
func (a *App) GoogleAuthHandler(c *gin.Context) {
	calendarConfig := InitGoogleCalendarConfig()
	if calendarConfig == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Google Calendar not configured"})
		return
	}

	// Generate state parameter for security
	state := fmt.Sprintf("user_%s_%d", c.Query("user_id"), time.Now().Unix())
	
	url := calendarConfig.Config.AuthCodeURL(state, oauth2.AccessTypeOffline)
	c.JSON(http.StatusOK, gin.H{
		"auth_url": url,
		"state":    state,
	})
}

// GoogleOAuth2CallbackHandler handles OAuth2 callback
func (a *App) GoogleOAuth2CallbackHandler(c *gin.Context) {
	calendarConfig := InitGoogleCalendarConfig()
	if calendarConfig == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Google Calendar not configured"})
		return
	}

	code := c.Query("code")
	state := c.Query("state")
	
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "authorization code required"})
		return
	}

	// Exchange code for token
	token, err := calendarConfig.Config.Exchange(context.Background(), code)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to exchange code for token"})
		return
	}

	// Store token (in a real app, you'd store this in database associated with user)
	tokenJSON, _ := json.Marshal(token)
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Authorization successful",
		"state":   state,
		"token":   string(tokenJSON), // In production, don't return token directly
	})
}

// GetGoogleCalendarEvents fetches events from Google Calendar
func (a *App) GetGoogleCalendarEvents(c *gin.Context) {
	// Get token from request (in production, get from database)
	tokenStr := c.GetHeader("X-Google-Token")
	if tokenStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Google token required in X-Google-Token header"})
		return
	}

	var token oauth2.Token
	if err := json.Unmarshal([]byte(tokenStr), &token); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid token format"})
		return
	}

	calendarConfig := InitGoogleCalendarConfig()
	if calendarConfig == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Google Calendar not configured"})
		return
	}

	// Create HTTP client with token
	client := calendarConfig.Config.Client(context.Background(), &token)
	
	// Create Calendar service
	srv, err := calendar.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create calendar service"})
		return
	}

	// Parse query parameters
	calendarID := c.DefaultQuery("calendar_id", "primary")
	timeMin := c.Query("time_min") // RFC3339 format
	timeMax := c.Query("time_max") // RFC3339 format
	maxResults := int64(250)

	// Build the events call
	eventsCall := srv.Events.List(calendarID).
		SingleEvents(true).
		OrderBy("startTime").
		MaxResults(maxResults)

	if timeMin != "" {
		eventsCall = eventsCall.TimeMin(timeMin)
	}
	if timeMax != "" {
		eventsCall = eventsCall.TimeMax(timeMax)
	}

	// Execute the call
	events, err := eventsCall.Do()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to retrieve events: %v", err)})
		return
	}

	// Convert to our format
	var calendarEvents []CalendarEvent
	for _, item := range events.Items {
		event := CalendarEvent{
			ID:          item.Id,
			Summary:     item.Summary,
			Description: item.Description,
			Location:    item.Location,
			Status:      item.Status,
		}

		// Handle creator
		if item.Creator != nil {
			event.Creator = item.Creator.Email
		}

		// Parse start time
		if item.Start.DateTime != "" {
			if startTime, err := time.Parse(time.RFC3339, item.Start.DateTime); err == nil {
				event.StartTime = startTime
			}
		} else if item.Start.Date != "" {
			if startTime, err := time.Parse("2006-01-02", item.Start.Date); err == nil {
				event.StartTime = startTime
			}
		}

		// Parse end time
		if item.End.DateTime != "" {
			if endTime, err := time.Parse(time.RFC3339, item.End.DateTime); err == nil {
				event.EndTime = endTime
			}
		} else if item.End.Date != "" {
			if endTime, err := time.Parse("2006-01-02", item.End.Date); err == nil {
				event.EndTime = endTime
			}
		}

		calendarEvents = append(calendarEvents, event)
	}

	c.JSON(http.StatusOK, gin.H{
		"events": calendarEvents,
		"count":  len(calendarEvents),
	})
}

// GetGoogleCalendarList fetches available calendars
func (a *App) GetGoogleCalendarList(c *gin.Context) {
	// Get token from request
	tokenStr := c.GetHeader("X-Google-Token")
	if tokenStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Google token required in X-Google-Token header"})
		return
	}

	var token oauth2.Token
	if err := json.Unmarshal([]byte(tokenStr), &token); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid token format"})
		return
	}

	calendarConfig := InitGoogleCalendarConfig()
	if calendarConfig == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Google Calendar not configured"})
		return
	}

	// Create HTTP client with token
	client := calendarConfig.Config.Client(context.Background(), &token)
	
	// Create Calendar service
	srv, err := calendar.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create calendar service"})
		return
	}

	// Get calendar list
	calendarList, err := srv.CalendarList.List().Do()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to retrieve calendars: %v", err)})
		return
	}

	type CalendarInfo struct {
		ID          string `json:"id"`
		Summary     string `json:"summary"`
		Description string `json:"description,omitempty"`
		Primary     bool   `json:"primary"`
		AccessRole  string `json:"access_role"`
	}

	var calendars []CalendarInfo
	for _, item := range calendarList.Items {
		calendar := CalendarInfo{
			ID:          item.Id,
			Summary:     item.Summary,
			Description: item.Description,
			Primary:     item.Primary,
			AccessRole:  item.AccessRole,
		}
		calendars = append(calendars, calendar)
	}

	c.JSON(http.StatusOK, gin.H{
		"calendars": calendars,
		"count":     len(calendars),
	})
}