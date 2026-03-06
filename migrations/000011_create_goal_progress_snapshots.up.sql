CREATE TABLE goal_progress_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    goal_id UUID NOT NULL REFERENCES saving_goals(id) ON DELETE CASCADE,
    amount DECIMAL(20, 2) NOT NULL,
    recorded_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_goal_progress_snapshots_goal_id ON goal_progress_snapshots(goal_id);
CREATE INDEX idx_goal_progress_snapshots_recorded_at ON goal_progress_snapshots(recorded_at);
