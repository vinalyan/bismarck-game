ALTER TABLE games
ADD COLUMN IF NOT EXISTS settings JSONB DEFAULT '{}'::jsonb;

UPDATE games
SET settings = '{}'::jsonb
WHERE settings IS NULL;

