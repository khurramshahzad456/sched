package app

import "time"

type AvailabilityRule struct {
	ID             int       `json:"id"`
	UserID         string    `json:"user_id"`
	DayOfWeek      int       `json:"day_of_week"`
	StartTime      string    `json:"start_time"`
	EndTime        string    `json:"end_time"`
	SlotLengthMins int       `json:"slot_length_minutes"`
	Title          string    `json:"title,omitempty"`
	Available      bool      `json:"available"`
	CreatedAt      time.Time `json:"created_at,omitempty"`
	UpdatedAt      time.Time `json:"updated_at,omitempty"`
}

type Booking struct {
	ID             string    `json:"id"`
	UserID         string    `json:"user_id"`
	CandidateEmail string    `json:"candidate_email"`
	StartAtUTC     time.Time `json:"start_at_utc"`
	EndAtUTC       time.Time `json:"end_at_utc"`
	Status         string    `json:"status"`
	Source         string    `json:"source,omitempty"`
	Type           string    `json:"type,omitempty"`
	Description    string    `json:"description,omitempty"`
	Title          string    `json:"title,omitempty"`
	CreatedAt      time.Time `json:"created_at,omitempty"`
}
