-- Tracks when a target's credentials were last set, so staleness can be
-- surfaced to operators (ListTargets computes credentialRotationDue from
-- this) instead of a static token being trusted forever with no visibility
-- into its age.
ALTER TABLE catalog_ai.ai_bridge_targets
    ADD COLUMN IF NOT EXISTS credentials_rotated_at TIMESTAMPTZ;
