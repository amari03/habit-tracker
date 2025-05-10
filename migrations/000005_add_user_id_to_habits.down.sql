-- Drop the index first if you created it
DROP INDEX IF EXISTS idx_habits_user_id;

-- Remove the foreign key constraint
ALTER TABLE habits
DROP CONSTRAINT IF EXISTS fk_habits_users;

-- Remove the user_id column
ALTER TABLE habits
DROP COLUMN IF EXISTS user_id;