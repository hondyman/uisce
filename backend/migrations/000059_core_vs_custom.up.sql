-- Core vs Custom Extensibility Schema

-- Add inheritance fields to semantic_cubes_v2
ALTER TABLE semantic_cubes_v2 ADD COLUMN IF NOT EXISTS source_cube_id UUID REFERENCES semantic_cubes_v2(id);
ALTER TABLE semantic_cubes_v2 ADD COLUMN IF NOT EXISTS is_system BOOLEAN DEFAULT false;

-- Add indexes for inheritance lookups
CREATE INDEX IF NOT EXISTS idx_semantic_cubes_v2_system ON semantic_cubes_v2(is_system);
CREATE INDEX IF NOT EXISTS idx_semantic_cubes_v2_source ON semantic_cubes_v2(source_cube_id);

-- Update comments
COMMENT ON COLUMN semantic_cubes_v2.source_cube_id IS 'Platform Core Cube ID that this Custom Cube extends';
COMMENT ON COLUMN semantic_cubes_v2.is_system IS 'True if this is a platform-provided Core Cube definition';
