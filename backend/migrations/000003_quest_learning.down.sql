ALTER TABLE quest_attempts
	DROP COLUMN IF EXISTS revealed_hints_count;

ALTER TABLE quests
	DROP COLUMN IF EXISTS real_world_importance,
	DROP COLUMN IF EXISTS glossary_terms,
	DROP COLUMN IF EXISTS after_solution_explanation,
	DROP COLUMN IF EXISTS progressive_hints;
