package app

import (
	"context"
	"fmt"
	"time"
)

// Slot DTO
type Slot struct {
	StartUTC time.Time `json:"start_utc"`
	EndUTC   time.Time `json:"end_utc"`
}

// generateSlots expands availability rules into slots in UTC between from/to inclusive,
// considering available=true rules and excluding available=false (unavailable) rules.
// It relies only on availability_rules table and existing bookings.
func (a *App) GenerateAvailableSlots(ctx context.Context, userID string, fromUTC, toUTC time.Time) ([]Slot, error) {
	// fetch user's rules
	rules, err := a.ListAvailabilityRules(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(rules) == 0 {
		return nil, nil
	}

	var candidateSlots []Slot

	// We'll iterate each date between fromUTC.Date and toUTC.Date in UTC
	startDate := fromUTC.Truncate(24 * time.Hour)
	endDate := toUTC.Truncate(24 * time.Hour)

	for day := startDate; !day.After(endDate); day = day.Add(24 * time.Hour) {
		for _, r := range rules {
			// Check if this day matches the rule's day of week (in UTC)
			if int(day.Weekday()) != r.DayOfWeek {
				continue
			}
			// parse start and end time (HH:MM) - now in UTC
			startTOD, err := parseHHMM(r.StartTime)
			if err != nil {
				return nil, err
			}
			endTOD, err := parseHHMM(r.EndTime)
			if err != nil {
				return nil, err
			}
			if !endTOD.After(startTOD) {
				return nil, fmt.Errorf("end_time must be after start_time for rule %d", r.ID)
			}
			// build UTC datetime
			year, month, dayNum := day.Date()
			utcStart := time.Date(year, month, dayNum, startTOD.Hour(), startTOD.Minute(), 0, 0, time.UTC)
			utcEnd := time.Date(year, month, dayNum, endTOD.Hour(), endTOD.Minute(), 0, 0, time.UTC)

			// chunk into slots
			slotLen := time.Duration(r.SlotLengthMins) * time.Minute
			for s := utcStart; s.Add(slotLen).Equal(utcEnd) || s.Add(slotLen).Before(utcEnd); s = s.Add(slotLen) {
				startUTC := s
				endUTC := s.Add(slotLen)
				if !endUTC.After(fromUTC) || !startUTC.Before(toUTC) {
					continue
				}
				if !r.Available {
					continue
				}
				candidateSlots = append(candidateSlots, Slot{StartUTC: startUTC, EndUTC: endUTC})
			}
		}
	}

	// remove slots that have confirmed bookings
	bookings, err := a.ListBookingsInRange(ctx, userID, fromUTC.Add(-1*time.Hour), toUTC.Add(1*time.Hour))
	if err != nil {
		return nil, err
	}
	bookedMap := map[int64]struct{}{}
	for _, b := range bookings {
		bookedMap[b.StartAtUTC.Unix()] = struct{}{}
	}

	var available []Slot
	for _, s := range candidateSlots {
		if _, ok := bookedMap[s.StartUTC.Unix()]; !ok {
			available = append(available, s)
		}
	}
	return available, nil
}

func parseHHMM(s string) (time.Time, error) {
	// Take first 5 chars "HH:MM"
	if len(s) < 5 {
		return time.Time{}, fmt.Errorf("invalid time string: %s", s)
	}
	s = s[:5] // "09:00:00.000000" -> "09:00"
	tt, err := time.Parse("15:04", s)
	if err != nil {
		return time.Time{}, err
	}
	return tt, nil
}
