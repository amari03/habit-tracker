ALTER TABLE habits
ADD COLUMN user_id INTEGER;

-- Add the foreign key constraint.
-- We make it NOT NULL later if all existing habits can be assigned a user
-- or if you decide new habits MUST have a user.
-- For now, let it be NULLABLE to avoid issues with existing data.
ALTER TABLE habits
ADD CONSTRAINT fk_habits_users
FOREIGN KEY (user_id)
REFERENCES users(id)
ON DELETE CASCADE; -- Or ON DELETE SET NULL / ON DELETE RESTRICT depending on desired behavior

-- Optional: If you have existing habits and a default user (e.g., an admin or a test user)
-- you might want to assign them to that user here.
-- For example, if user with id 1 is your default/test user:
-- UPDATE habits SET user_id = 1 WHERE user_id IS NULL;

-- After potentially populating existing habits, if you decide user_id MUST NOT be NULL
-- for all new and existing habits, you can add the NOT NULL constraint.
-- Be cautious with this if you have habits you can't assign to a user.
-- ALTER TABLE habits
-- ALTER COLUMN user_id SET NOT NULL;

-- Optional: Add an index for performance on lookups by user_id
CREATE INDEX IF NOT EXISTS idx_habits_user_id ON habits (user_id);