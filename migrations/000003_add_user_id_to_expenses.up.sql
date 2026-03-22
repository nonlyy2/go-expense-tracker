ALTER TABLE expenses ADD COLUMN user_id INTEGER REFERENCES users(id);
CREATE INDEX idx_expenses_user_id ON expenses(user_id);