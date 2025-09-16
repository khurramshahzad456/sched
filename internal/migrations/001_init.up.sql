CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE TABLE IF NOT EXISTS availability_rules (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL,
    day_of_week INT NOT NULL CHECK (day_of_week >= 0 AND day_of_week <= 6),
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    slot_length_minutes INT NOT NULL CHECK (slot_length_minutes > 0),
    timezone TEXT NOT NULL,
    available BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS bookings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    candidate_email TEXT NOT NULL,
    start_at_utc TIMESTAMPTZ NOT NULL,
    end_at_utc TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL DEFAULT 'confirmed',
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_bookings_user_start_confirmed
    ON bookings (user_id, start_at_utc)
    WHERE status = 'confirmed';
