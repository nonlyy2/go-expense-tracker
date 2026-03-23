ALTER TABLE expenses ADD COLUMN IF NOT EXISTS user_id INT REFERENCES users(id) ON DELETE CASCADE;
CREATE INDEX IF NOT EXISTS idx_expenses_user_id ON expenses(user_id);