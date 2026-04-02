-- 002_model_performance.sql
-- Adds index for faster model+task_type lookups (table already created in 001)

CREATE INDEX IF NOT EXISTS idx_model_performance_model_task ON model_performance(model, task_type);
