ALTER TABLE availability_rules
ADD CONSTRAINT uniq_user_day UNIQUE (user_id, day_of_week);
