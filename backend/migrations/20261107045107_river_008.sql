-- migrate:up

-- River main migration 008 [up]
-- No-op. Postgres sequences already prevent automatically generated job IDs
-- from being reused.
SELECT 1;

-- migrate:down

-- River main migration 008 [down]
-- No-op. Postgres sequences already prevent automatically generated job IDs
-- from being reused.
SELECT 1;
