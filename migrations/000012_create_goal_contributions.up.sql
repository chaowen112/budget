CREATE TABLE goal_contributions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    goal_id UUID NOT NULL REFERENCES saving_goals(id) ON DELETE CASCADE,
    amount_delta DECIMAL(20, 2) NOT NULL,
    balance_after DECIMAL(20, 2) NOT NULL,
    source TEXT NOT NULL DEFAULT 'manual',
    recorded_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_goal_contributions_goal_id ON goal_contributions(goal_id);
CREATE INDEX idx_goal_contributions_recorded_at ON goal_contributions(recorded_at);

INSERT INTO goal_contributions (goal_id, amount_delta, balance_after, source, recorded_at)
SELECT
    s.goal_id,
    COALESCE(
        s.amount - LAG(s.amount) OVER (PARTITION BY s.goal_id ORDER BY s.recorded_at, s.id),
        s.amount
    ) AS amount_delta,
    s.amount AS balance_after,
    'backfill_snapshot' AS source,
    s.recorded_at
FROM goal_progress_snapshots s;
