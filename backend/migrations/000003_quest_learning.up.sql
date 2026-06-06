ALTER TABLE quests
	ADD COLUMN progressive_hints jsonb NOT NULL DEFAULT '[]'::jsonb,
	ADD COLUMN after_solution_explanation text NOT NULL DEFAULT '',
	ADD COLUMN glossary_terms jsonb NOT NULL DEFAULT '[]'::jsonb,
	ADD COLUMN real_world_importance text NOT NULL DEFAULT '';

ALTER TABLE quest_attempts
	ADD COLUMN revealed_hints_count integer NOT NULL DEFAULT 0;
