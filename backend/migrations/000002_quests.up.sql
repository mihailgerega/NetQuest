CREATE TABLE quests (
	id text PRIMARY KEY,
	slug text UNIQUE NOT NULL,
	title text NOT NULL,
	difficulty text NOT NULL CHECK (difficulty IN ('easy', 'medium', 'hard')),
	category text NOT NULL,
	description text NOT NULL,
	goal text NOT NULL,
	learning_objectives jsonb NOT NULL DEFAULT '[]'::jsonb,
	initial_topology jsonb NOT NULL,
	expected_checks jsonb NOT NULL DEFAULT '[]'::jsonb,
	hints jsonb NOT NULL DEFAULT '[]'::jsonb,
	success_message text NOT NULL,
	failure_message text NOT NULL,
	estimated_minutes integer NOT NULL DEFAULT 10,
	created_at timestamptz NOT NULL DEFAULT now(),
	updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TRIGGER quests_set_updated_at
BEFORE UPDATE ON quests
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE quest_attempts (
	id uuid PRIMARY KEY,
	quest_id text NOT NULL REFERENCES quests(id) ON DELETE CASCADE,
	user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	project_id uuid REFERENCES projects(id) ON DELETE SET NULL,
	current_topology_id uuid REFERENCES topologies(id) ON DELETE SET NULL,
	status text NOT NULL CHECK (status IN ('not_started', 'in_progress', 'completed', 'failed')),
	attempts_count integer NOT NULL DEFAULT 0,
	last_check_result jsonb,
	completed_at timestamptz,
	created_at timestamptz NOT NULL DEFAULT now(),
	updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_quest_attempts_user_id ON quest_attempts(user_id);
CREATE INDEX idx_quest_attempts_quest_user ON quest_attempts(quest_id, user_id);
CREATE INDEX idx_quest_attempts_status ON quest_attempts(status);

CREATE TRIGGER quest_attempts_set_updated_at
BEFORE UPDATE ON quest_attempts
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE quest_check_results (
	id uuid PRIMARY KEY,
	attempt_id uuid NOT NULL REFERENCES quest_attempts(id) ON DELETE CASCADE,
	user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	passed boolean NOT NULL,
	score integer NOT NULL,
	result jsonb NOT NULL,
	created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_quest_check_results_attempt_id ON quest_check_results(attempt_id);
