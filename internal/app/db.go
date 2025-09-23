package app

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

func (a *App) InsertAvailabilityRule(ctx context.Context, r *AvailabilityRule) error {
	now := time.Now().UTC()

	// Insert - no uniqueness check, allow multiple rules per day
	q := `INSERT INTO availability_rules
          (id, user_id, day_of_week, start_time, end_time, slot_length_minutes, title, available, created_at, updated_at)
          VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`

	row := a.DB.QueryRow(ctx, q,
		r.UserID, r.DayOfWeek, r.StartTime, r.EndTime, r.SlotLengthMins,
		r.Title, r.Available, now, now)

	return row.Scan(&r.ID)
}

func (a *App) ListAvailabilityRules(ctx context.Context, userID string) ([]AvailabilityRule, error) {
	q := `SELECT id,user_id,day_of_week,start_time,end_time,slot_length_minutes,title,available,created_at,updated_at
	      FROM availability_rules WHERE user_id=$1 ORDER BY id`
	rows, err := a.DB.Query(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []AvailabilityRule
	for rows.Next() {
		var r AvailabilityRule
		var start, end string
		if err := rows.Scan(&r.ID, &r.UserID, &r.DayOfWeek, &start, &end,
			&r.SlotLengthMins, &r.Title, &r.Available, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		r.StartTime = start
		r.EndTime = end
		out = append(out, r)
	}
	return out, nil
}

func (a *App) ListBookingsInRange(ctx context.Context, userID string, from, to time.Time) ([]Booking, error) {
	q := `SELECT id,user_id,candidate_email,start_at_utc,end_at_utc,status,created_at 
	      FROM bookings
	      WHERE user_id=$1 AND start_at_utc >= $2 AND start_at_utc < $3 AND status='confirmed'`
	rows, err := a.DB.Query(ctx, q, userID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Booking
	for rows.Next() {
		var b Booking
		if err := rows.Scan(&b.ID, &b.UserID, &b.CandidateEmail,
			&b.StartAtUTC, &b.EndAtUTC, &b.Status, &b.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, nil
}

func (a *App) ListBookings(ctx context.Context, userID string, from, to time.Time, filtered bool) ([]Booking, error) {
	var (
		rows pgx.Rows
		err  error
	)

	if filtered {
		q := `SELECT id,user_id,candidate_email,start_at_utc,end_at_utc,status,created_at 
              FROM bookings 
              WHERE user_id=$1 AND start_at_utc >= $2 AND start_at_utc < $3 AND status != 'cancelled'
              ORDER BY start_at_utc`
		rows, err = a.DB.Query(ctx, q, userID, from, to)
	} else {
		q := `SELECT id,user_id,candidate_email,start_at_utc,end_at_utc,status,created_at 
              FROM bookings 
              WHERE user_id=$1 AND status != 'cancelled'
              ORDER BY start_at_utc`
		rows, err = a.DB.Query(ctx, q, userID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Booking
	for rows.Next() {
		var b Booking
		if err := rows.Scan(&b.ID, &b.UserID, &b.CandidateEmail, &b.StartAtUTC, &b.EndAtUTC, &b.Status, &b.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, nil
}
