-- Add is_patrolling field to task_forces (for patrol markers +3 search factors)
ALTER TABLE task_forces
ADD COLUMN IF NOT EXISTS is_patrolling BOOLEAN DEFAULT FALSE;

-- Create index on is_patrolling for faster queries
CREATE INDEX IF NOT EXISTS idx_task_forces_is_patrolling ON task_forces(is_patrolling) WHERE is_patrolling = true;

-- Add comment to column
COMMENT ON COLUMN task_forces.is_patrolling IS 'Indicates if the task force is patrolling (gives +3 search factors in its hex)';

