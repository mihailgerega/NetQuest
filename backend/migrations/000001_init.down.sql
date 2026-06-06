DROP INDEX IF EXISTS idx_audit_logs_resource;
DROP INDEX IF EXISTS idx_audit_logs_action;
DROP INDEX IF EXISTS idx_audit_logs_user_id;
DROP TABLE IF EXISTS audit_logs;

DROP INDEX IF EXISTS idx_simulation_events_type;
DROP INDEX IF EXISTS idx_simulation_events_simulation_id;
DROP TABLE IF EXISTS simulation_events;

DROP INDEX IF EXISTS idx_simulations_status;
DROP INDEX IF EXISTS idx_simulations_user_id;
DROP INDEX IF EXISTS idx_simulations_topology_id;
DROP INDEX IF EXISTS idx_simulations_project_id;
DROP TABLE IF EXISTS simulations;

DROP INDEX IF EXISTS idx_topologies_project_version_active;
DROP INDEX IF EXISTS idx_topologies_project_id;
DROP TABLE IF EXISTS topologies;

DROP INDEX IF EXISTS idx_projects_visibility;
DROP INDEX IF EXISTS idx_projects_owner_id;
DROP TABLE IF EXISTS projects;

DROP INDEX IF EXISTS idx_refresh_tokens_user_id;
DROP TABLE IF EXISTS refresh_tokens;

DROP TABLE IF EXISTS users;
DROP FUNCTION IF EXISTS set_updated_at();
