CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS trigger AS $$
BEGIN
	NEW.updated_at = now();
	RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE users (
	id uuid PRIMARY KEY,
	email text UNIQUE NOT NULL,
	password_hash text,
	display_name text,
	avatar_url text,
	role text NOT NULL CHECK (role IN ('guest', 'user', 'teacher', 'admin')),
	created_at timestamptz NOT NULL DEFAULT now(),
	updated_at timestamptz NOT NULL DEFAULT now(),
	deleted_at timestamptz
);

CREATE TRIGGER users_set_updated_at
BEFORE UPDATE ON users
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE refresh_tokens (
	id uuid PRIMARY KEY,
	user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	token_hash text NOT NULL,
	user_agent text,
	ip_address inet,
	expires_at timestamptz NOT NULL,
	revoked_at timestamptz,
	created_at timestamptz NOT NULL DEFAULT now(),
	updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);

CREATE TRIGGER refresh_tokens_set_updated_at
BEFORE UPDATE ON refresh_tokens
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE projects (
	id uuid PRIMARY KEY,
	owner_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
	name text NOT NULL,
	description text,
	visibility text NOT NULL CHECK (visibility IN ('private', 'public', 'unlisted')),
	created_at timestamptz NOT NULL DEFAULT now(),
	updated_at timestamptz NOT NULL DEFAULT now(),
	deleted_at timestamptz
);

CREATE INDEX idx_projects_owner_id ON projects(owner_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_projects_visibility ON projects(visibility) WHERE deleted_at IS NULL;

CREATE TRIGGER projects_set_updated_at
BEFORE UPDATE ON projects
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE topologies (
	id uuid PRIMARY KEY,
	project_id uuid NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
	version integer NOT NULL,
	name text NOT NULL,
	data jsonb NOT NULL,
	created_at timestamptz NOT NULL DEFAULT now(),
	updated_at timestamptz NOT NULL DEFAULT now(),
	deleted_at timestamptz,
	created_by uuid REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX idx_topologies_project_id ON topologies(project_id) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX idx_topologies_project_version_active ON topologies(project_id, version) WHERE deleted_at IS NULL;

CREATE TRIGGER topologies_set_updated_at
BEFORE UPDATE ON topologies
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE simulations (
	id uuid PRIMARY KEY,
	project_id uuid NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
	topology_id uuid NOT NULL REFERENCES topologies(id) ON DELETE RESTRICT,
	user_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
	status text NOT NULL CHECK (status IN ('pending', 'running', 'completed', 'failed')),
	scenario jsonb NOT NULL,
	seed bigint NOT NULL,
	started_at timestamptz,
	finished_at timestamptz,
	error_message text,
	created_at timestamptz NOT NULL DEFAULT now(),
	updated_at timestamptz NOT NULL DEFAULT now(),
	deleted_at timestamptz
);

CREATE INDEX idx_simulations_project_id ON simulations(project_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_simulations_topology_id ON simulations(topology_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_simulations_user_id ON simulations(user_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_simulations_status ON simulations(status) WHERE deleted_at IS NULL;

CREATE TRIGGER simulations_set_updated_at
BEFORE UPDATE ON simulations
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE simulation_events (
	id uuid PRIMARY KEY,
	simulation_id uuid NOT NULL REFERENCES simulations(id) ON DELETE CASCADE,
	sequence_number bigint NOT NULL,
	timestamp_ms bigint NOT NULL,
	type text NOT NULL,
	severity text NOT NULL CHECK (severity IN ('info', 'warn', 'error')),
	packet_id text,
	source_node_id text,
	target_node_id text,
	message text NOT NULL,
	details jsonb NOT NULL DEFAULT '{}'::jsonb,
	created_at timestamptz NOT NULL DEFAULT now(),
	updated_at timestamptz NOT NULL DEFAULT now(),
	UNIQUE (simulation_id, sequence_number)
);

CREATE INDEX idx_simulation_events_simulation_id ON simulation_events(simulation_id);
CREATE INDEX idx_simulation_events_type ON simulation_events(type);

CREATE TABLE audit_logs (
	id uuid PRIMARY KEY,
	user_id uuid REFERENCES users(id) ON DELETE SET NULL,
	action text NOT NULL,
	resource_type text,
	resource_id text,
	ip_address inet,
	user_agent text,
	metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
	created_at timestamptz NOT NULL DEFAULT now(),
	updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_resource ON audit_logs(resource_type, resource_id);
