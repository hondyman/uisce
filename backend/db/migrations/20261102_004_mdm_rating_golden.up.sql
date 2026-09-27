-- 20261102_004_mdm_rating_golden.up.sql
-- Golden / survivorship views for the rating domain.

-- ── golden_rating: current, active, highest-rank rating per party ──────
CREATE OR REPLACE VIEW mdm.golden_rating AS
SELECT DISTINCT ON (r.tenant_id, r.rated_party_type, r.rated_party_id, r.rating_type_id)
    r.id                          AS rating_id,
    r.tenant_id,
    r.rated_party_type,
    r.rated_party_id,
    r.rated_party_key,
    r.rating_type_id,
    r.agency_id,
    ag.agency_cd,
    r.rating_scale_id,
    r.rating_value,
    r.rating_rank,
    r.outlook_id,
    o.outlook_cd,
    r.watch_id,
    w.watch_cd,
    r.credit_watch_direction,
    r.rating_date,
    r.effective_from,
    r.created_at,
    r.updated_at
FROM mdm.rating r
JOIN mdm.rating_agency ag ON ag.id = r.agency_id
LEFT JOIN mdm.rating_outlook o ON o.id = r.outlook_id
LEFT JOIN mdm.rating_watch   w ON w.id = r.watch_id
WHERE r.is_active
  AND r.is_latest
  AND r.effective_to IS NULL
ORDER BY
    r.tenant_id,
    r.rated_party_type,
    r.rated_party_id,
    r.rating_type_id,
    r.rating_rank ASC NULLS LAST,   -- rank 1 = best
    r.rating_date DESC,
    r.created_at DESC;

-- ── golden_rating_agency: one best rating per party per agency ─────────
CREATE OR REPLACE VIEW mdm.golden_rating_agency AS
SELECT DISTINCT ON (r.tenant_id, r.rated_party_type, r.rated_party_id, r.agency_id)
    r.id                     AS rating_id,
    r.tenant_id,
    r.rated_party_type,
    r.rated_party_id,
    r.agency_id,
    ag.agency_cd,
    ag.short_name            AS agency_short_name,
    r.rating_type_id,
    rt.type_cd               AS rating_type_cd,
    r.rating_value,
    r.rating_rank,
    r.outlook_id,
    o.outlook_cd,
    r.rating_date
FROM mdm.rating r
JOIN mdm.rating_agency ag  ON ag.id = r.agency_id
JOIN mdm.rating_type   rt  ON rt.id = r.rating_type_id
LEFT JOIN mdm.rating_outlook o ON o.id = r.outlook_id
WHERE r.is_active
  AND r.is_latest
  AND r.effective_to IS NULL
ORDER BY
    r.tenant_id,
    r.rated_party_type,
    r.rated_party_id,
    r.agency_id,
    r.rating_rank ASC NULLS LAST,
    r.rating_date DESC;

-- ── golden_rating_summary: condensed row for UI / API ──────────────────
CREATE OR REPLACE VIEW mdm.golden_rating_summary AS
WITH best AS (
    SELECT * FROM mdm.golden_rating_agency
),
worst AS (
    SELECT DISTINCT ON (tenant_id, rated_party_type, rated_party_id)
        tenant_id, rated_party_type, rated_party_id,
        rating_value AS worst_value, rating_rank AS worst_rank,
        agency_cd    AS worst_agency_cd, outlook_cd AS worst_outlook_cd
    FROM best
    ORDER BY tenant_id, rated_party_type, rated_party_id,
             rating_rank DESC NULLS LAST
),
watch_any AS (
    SELECT DISTINCT ON (tenant_id, rated_party_type, rated_party_id)
        tenant_id, rated_party_type, rated_party_id,
        true AS on_watch
    FROM mdm.rating
    WHERE is_active AND is_latest AND effective_to IS NULL
      AND (credit_watch_direction IS NOT NULL OR watch_id IS NOT NULL)
)
SELECT
    b.tenant_id,
    b.rated_party_type,
    b.rated_party_id,
    COUNT(*)                                              AS agency_count,
    MIN(b.rating_rank)                                    AS best_rank,
    w.worst_value                                         AS worst_value,
    w.worst_agency_cd                                     AS worst_agency_cd,
    w.worst_outlook_cd                                    AS worst_outlook_cd,
    COALESCE(wa.on_watch, false)                          AS on_watch,
    MAX(b.rating_date)                                    AS latest_rating_date
FROM best b
JOIN worst w
  ON w.tenant_id = b.tenant_id
 AND w.rated_party_type = b.rated_party_type
 AND w.rated_party_id = b.rated_party_id
LEFT JOIN watch_any wa
  ON wa.tenant_id = b.tenant_id
 AND wa.rated_party_type = b.rated_party_type
 AND wa.rated_party_id = b.rated_party_id
GROUP BY
    b.tenant_id, b.rated_party_type, b.rated_party_id,
    w.worst_value, w.worst_agency_cd, w.worst_outlook_cd, wa.on_watch;

-- ── golden_internal_rating: current house rating per party ─────────────
CREATE OR REPLACE VIEW mdm.golden_internal_rating AS
SELECT DISTINCT ON (tenant_id, rated_party_type, rated_party_id, model_name)
    id,
    tenant_id,
    rated_party_type,
    rated_party_id,
    internal_scale_cd,
    internal_value,
    internal_rank,
    score,
    score_band,
    model_name,
    model_version,
    analyst_id,
    rationale,
    next_review_date,
    effective_from
FROM mdm.rating_internal
WHERE is_active
  AND effective_to IS NULL
ORDER BY
    tenant_id, rated_party_type, rated_party_id, model_name,
    internal_rank ASC NULLS LAST,
    effective_from DESC;

-- ── rating_divergence: proprietary vs consensus ────────────────────────
CREATE OR REPLACE VIEW mdm.rating_divergence AS
SELECT
    rp.tenant_id,
    rp.rated_party_type,
    rp.rated_party_id,
    rp.prop_value,
    rp.prop_rank,
    gr.rating_value  AS consensus_value,
    gr.rating_rank   AS consensus_rank,
    gr.agency_cd     AS consensus_agency_cd,
    (rp.prop_rank - gr.rating_rank) AS rank_divergence,
    rp.model_cd,
    rp.input_as_of
FROM mdm.rating_proprietary rp
JOIN mdm.golden_rating gr
  ON gr.tenant_id        = rp.tenant_id
 AND gr.rated_party_type = rp.rated_party_type
 AND gr.rated_party_id   = rp.rated_party_id
WHERE rp.is_current;
