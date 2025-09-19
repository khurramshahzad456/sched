package app

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

// POST /users/:id/availability
// Accepts list of rules (create/update by id if provided).
func (a *App) SetAvailabilityHandler(c *gin.Context) {
	userID := c.Param("id")
	var payload []AvailabilityRule
	if err := c.BindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := c.Request.Context()

	var savedRules []AvailabilityRule
	for i := range payload {
		payload[i].UserID = userID
		if err := a.InsertAvailabilityRule(ctx, &payload[i]); err != nil {
			if strings.Contains(err.Error(), "already exists") {
				c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		savedRules = append(savedRules, payload[i])
	}

	c.JSON(http.StatusCreated, savedRules)
}

// PUT /users/:id/availability/:rule_id
func (a *App) UpdateAvailabilityHandler(c *gin.Context) {
	userID := c.Param("id")
	ruleID := c.Param("rule_id")

	var payload AvailabilityRule
	if err := c.BindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	now := time.Now().UTC()

	q := `UPDATE availability_rules
          SET start_time=$1, end_time=$2, slot_length_minutes=$3,
              timezone=$4, available=$5, updated_at=$6
          WHERE id=$7 AND user_id=$8
          RETURNING id`

	var updatedID int
	err := a.DB.QueryRow(ctx, q,
		payload.StartTime, payload.EndTime, payload.SlotLengthMins,
		payload.Timezone, payload.Available, now, ruleID, userID,
	).Scan(&updatedID)

	if err == pgx.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "availability not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	payload.ID = updatedID
	payload.UserID = userID
	payload.UpdatedAt = now

	c.JSON(http.StatusOK, payload)
}

// GET /users/:id/availability
func (a *App) ListAvailabilityHandler(c *gin.Context) {
	userID := c.Param("id")
	rules, err := a.ListAvailabilityRules(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rules)
}

// GET /users/:id/slots?from=ISO&to=ISO
func (a *App) GetSlotsHandler(c *gin.Context) {
	userID := c.Param("id")
	fromStr := c.Query("from")
	toStr := c.Query("to")
	if fromStr == "" || toStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "from and to required (ISO8601)"})
		return
	}
	from, err := time.Parse(time.RFC3339, fromStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid from"})
		return
	}
	to, err := time.Parse(time.RFC3339, toStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid to"})
		return
	}
	if !from.Before(to) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "from must be before to"})
		return
	}
	slots, err := a.GenerateAvailableSlots(c.Request.Context(), userID, from.UTC(), to.UTC())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, slots)
}

type createBookingReq struct {
	UserID         string `json:"user_id"` // optional, since it's in URL path
	CandidateEmail string `json:"candidate_email" binding:"required,email"`
	StartAtUTCStr  string `json:"start_at_utc" binding:"required"` // RFC3339
	EndAtUTCStr    string `json:"end_at_utc" binding:"required"`
	Source         string `json:"source,omitempty"`
	Type           string `json:"type,omitempty"`
	Description    string `json:"description,omitempty"`
	Title          string `json:"title,omitempty"`
}

// GET /users/:id/bookings?from=ISO&to=ISO
func (a *App) ListBookingsHandler(c *gin.Context) {
	userID := c.Param("id")
	fromStr := c.Query("from")
	toStr := c.Query("to")

	ctx := c.Request.Context()

	var (
		from time.Time
		to   time.Time
		err  error
	)

	// if both provided, parse
	if fromStr != "" && toStr != "" {
		from, err = time.Parse(time.RFC3339, fromStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid from"})
			return
		}
		to, err = time.Parse(time.RFC3339, toStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid to"})
			return
		}
		if !from.Before(to) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "from must be before to"})
			return
		}
	}

	bookings, err := a.ListBookings(ctx, userID, from, to, fromStr != "" && toStr != "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, bookings)
}

// POST /users/:id/bookings
func (a *App) CreateBookingHandler(c *gin.Context) {
	userID := c.Param("id")
	var req createBookingReq
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	start, err := time.Parse(time.RFC3339, req.StartAtUTCStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start_at_utc"})
		return
	}
	end, err := time.Parse(time.RFC3339, req.EndAtUTCStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end_at_utc"})
		return
	}
	if !start.Before(end) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "start must be before end"})
		return
	}

	ctx := context.Background()
	tx, err := a.DB.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer tx.Rollback(ctx)

	// check overlapping confirmed booking
	checkQ := `SELECT id FROM bookings 
			   WHERE user_id=$1 AND status='confirmed' 
			   AND start_at_utc = $2 FOR UPDATE`
	var existingID string
	err = tx.QueryRow(ctx, checkQ, userID, start.UTC()).Scan(&existingID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if existingID != "" {
		c.JSON(http.StatusConflict, gin.H{"error": "slot already booked"})
		return
	}

	// verify slot belongs to user's availability
	slots, err := a.GenerateAvailableSlots(ctx, userID, start.Add(-1*time.Second).UTC(), end.Add(1*time.Second).UTC())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ok := false
	for _, s := range slots {
		if s.StartUTC.Equal(start.UTC()) && s.EndUTC.Equal(end.UTC()) {
			ok = true
			break
		}
	}
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "slot not available"})
		return
	}

	// insert booking
	insertQ := `INSERT INTO bookings 
		(id, user_id, candidate_email, start_at_utc, end_at_utc, status, source, type, description, title, created_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, 'confirmed', $5, $6, $7, $8, now())
		RETURNING id`
	var newID string
	err = tx.QueryRow(
		ctx, insertQ,
		userID,
		req.CandidateEmail,
		start.UTC(),
		end.UTC(),
		req.Source,
		req.Type,
		req.Description,
		req.Title,
	).Scan(&newID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":           newID,
		"status":       "confirmed",
		"start_at_utc": start.UTC(),
		"end_at_utc":   end.UTC(),
		"source":       req.Source,
		"type":         req.Type,
		"description":  req.Description,
		"title":        req.Title,
	})
}

// DELETE /bookings/:id
func (a *App) CancelBookingHandler(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()

	// First check if the booking exists and get its current status
	checkQ := `SELECT status FROM bookings WHERE id=$1`
	var currentStatus string
	err := a.DB.QueryRow(ctx, checkQ, id).Scan(&currentStatus)
	if err == pgx.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "booking not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Check if already cancelled
	if currentStatus == "cancelled" {
		c.JSON(http.StatusConflict, gin.H{"error": "booking not found"})
		return
	}

	// Update to cancelled
	updateQ := `UPDATE bookings SET status='cancelled' WHERE id=$1 AND status != 'cancelled'`
	res, err := a.DB.Exec(ctx, updateQ, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if res.RowsAffected() == 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "booking not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}
