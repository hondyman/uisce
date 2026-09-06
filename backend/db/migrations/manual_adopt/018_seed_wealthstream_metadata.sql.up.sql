-- Seed WealthStream Product Layer Metadata
-- Implements the "WealthStream" vision: AI Feed, Personas, Advisor Escalation

-- 1. Client Understanding & Personalization
-- Client Profile (Extends basic client with behavioral data)
INSERT INTO meta_objects (id, name, version_major, version_minor, version_patch, status, payload)
VALUES ('bo_client_profile_extended', 'Client Profile (Extended)', 1, 0, 0, 'active', '{
  "meta": {
    "id": "bo_client_profile_extended",
    "name": "Client Profile (Extended)",
    "version": {"major": 1, "minor": 0, "patch": 0},
    "status": "active",
    "valid_from": "2025-01-01T00:00:00Z"
  },
  "attributes": [
    {"name": "client_id", "type": "string", "required": true},
    {"name": "risk_tolerance_score", "type": "integer", "required": true, "description": "1-100 score"},
    {"name": "esg_focus_areas", "type": "array<string>", "required": false},
    {"name": "behavior_tags", "type": "array<string>", "required": false, "description": "e.g., anxious, hands-off"},
    {"name": "last_login_date", "type": "datetime", "required": false}
  ],
  "rels": [
    {"name": "client", "target_bo": "bo_client", "cardinality": "one", "on_delete": "cascade"}
  ],
  "lifecycle": ["ACTIVE", "ARCHIVED"],
  "policies": ["VIEW_SENSITIVE_DATA"]
}');

-- Life Event
INSERT INTO meta_objects (id, name, version_major, version_minor, version_patch, status, payload)
VALUES ('bo_life_event', 'Life Event', 1, 0, 0, 'active', '{
  "meta": {
    "id": "bo_life_event",
    "name": "Life Event",
    "version": {"major": 1, "minor": 0, "patch": 0},
    "status": "active"
  },
  "attributes": [
    {"name": "event_type", "type": "string", "required": true, "description": "MARRIAGE, JOB_CHANGE, RELOCATION"},
    {"name": "event_date", "type": "date", "required": true},
    {"name": "description", "type": "string", "required": false},
    {"name": "impact_score", "type": "integer", "required": false}
  ],
  "rels": [
    {"name": "client", "target_bo": "bo_client", "cardinality": "one", "on_delete": "cascade"}
  ],
  "lifecycle": ["DETECTED", "CONFIRMED", "ACTIONED", "DISMISSED"]
}');

-- Engagement Preference
INSERT INTO meta_objects (id, name, version_major, version_minor, version_patch, status, payload)
VALUES ('bo_engagement_preference', 'Engagement Preference', 1, 0, 0, 'active', '{
  "meta": {
    "id": "bo_engagement_preference",
    "name": "Engagement Preference",
    "version": {"major": 1, "minor": 0, "patch": 0},
    "status": "active"
  },
  "attributes": [
    {"name": "preferred_channel", "type": "string", "required": true, "default": "MOBILE_PUSH"},
    {"name": "frequency", "type": "string", "required": true, "default": "WEEKLY"},
    {"name": "quiet_hours_start", "type": "time", "required": false},
    {"name": "quiet_hours_end", "type": "time", "required": false},
    {"name": "tone_preference", "type": "string", "required": false, "default": "PROFESSIONAL"}
  ],
  "rels": [
    {"name": "client", "target_bo": "bo_client", "cardinality": "one", "on_delete": "cascade"}
  ]
}');

-- 2. AI Specialist Insights & Actions
-- Insight
INSERT INTO meta_objects (id, name, version_major, version_minor, version_patch, status, payload)
VALUES ('bo_insight', 'Insight', 1, 0, 0, 'active', '{
  "meta": {
    "id": "bo_insight",
    "name": "Insight",
    "version": {"major": 1, "minor": 0, "patch": 0},
    "status": "active"
  },
  "attributes": [
    {"name": "source_model", "type": "string", "required": true},
    {"name": "insight_type", "type": "string", "required": true, "description": "PLANNING, MARKET, OPERATIONS"},
    {"name": "confidence_score", "type": "decimal", "required": true},
    {"name": "summary", "type": "string", "required": true},
    {"name": "details", "type": "jsonb", "required": false},
    {"name": "generated_at", "type": "datetime", "required": true}
  ],
  "rels": [
    {"name": "client", "target_bo": "bo_client", "cardinality": "one", "on_delete": "cascade"},
    {"name": "related_asset", "target_bo": "bo_asset", "cardinality": "zero_or_one"}
  ],
  "lifecycle": ["NEW", "VIEWED", "ACTED_ON", "DISMISSED"]
}');

-- Recommendation
INSERT INTO meta_objects (id, name, version_major, version_minor, version_patch, status, payload)
VALUES ('bo_recommendation', 'Recommendation', 1, 0, 0, 'active', '{
  "meta": {
    "id": "bo_recommendation",
    "name": "Recommendation",
    "version": {"major": 1, "minor": 0, "patch": 0},
    "status": "active"
  },
  "attributes": [
    {"name": "action_type", "type": "string", "required": true},
    {"name": "description", "type": "string", "required": true},
    {"name": "priority", "type": "string", "required": true, "default": "MEDIUM"},
    {"name": "prerequisites", "type": "array<string>", "required": false},
    {"name": "requires_approval", "type": "boolean", "required": true, "default": false}
  ],
  "rels": [
    {"name": "insight", "target_bo": "bo_insight", "cardinality": "one", "on_delete": "cascade"}
  ],
  "lifecycle": ["PROPOSED", "APPROVED", "REJECTED", "EXECUTED"]
}');

-- Alert Policy
INSERT INTO meta_objects (id, name, version_major, version_minor, version_patch, status, payload)
VALUES ('bo_alert_policy', 'Alert Policy', 1, 0, 0, 'active', '{
  "meta": {
    "id": "bo_alert_policy",
    "name": "Alert Policy",
    "version": {"major": 1, "minor": 0, "patch": 0},
    "status": "active"
  },
  "attributes": [
    {"name": "policy_name", "type": "string", "required": true},
    {"name": "trigger_condition", "type": "string", "required": true, "description": "CEL expression"},
    {"name": "throttle_limit", "type": "integer", "required": false},
    {"name": "throttle_window_hours", "type": "integer", "required": false},
    {"name": "eligibility_criteria", "type": "string", "required": false}
  ],
  "lifecycle": ["DRAFT", "ACTIVE", "PAUSED"]
}');

-- 3. Personalized Feed & Omnichannel
-- Feed Card
INSERT INTO meta_objects (id, name, version_major, version_minor, version_patch, status, payload)
VALUES ('bo_feed_card', 'Feed Card', 1, 0, 0, 'active', '{
  "meta": {
    "id": "bo_feed_card",
    "name": "Feed Card",
    "version": {"major": 1, "minor": 0, "patch": 0},
    "status": "active"
  },
  "attributes": [
    {"name": "template_id", "type": "string", "required": true},
    {"name": "title", "type": "string", "required": true},
    {"name": "body", "type": "string", "required": true},
    {"name": "image_url", "type": "string", "required": false},
    {"name": "action_label", "type": "string", "required": false},
    {"name": "action_url", "type": "string", "required": false},
    {"name": "priority_score", "type": "decimal", "required": true},
    {"name": "tone", "type": "string", "required": false}
  ],
  "rels": [
    {"name": "client", "target_bo": "bo_client", "cardinality": "one", "on_delete": "cascade"},
    {"name": "insight", "target_bo": "bo_insight", "cardinality": "zero_or_one"}
  ],
  "lifecycle": ["GENERATED", "DELIVERED", "CLICKED", "DISMISSED", "EXPIRED"]
}');

-- Channel
INSERT INTO meta_objects (id, name, version_major, version_minor, version_patch, status, payload)
VALUES ('bo_channel', 'Channel', 1, 0, 0, 'active', '{
  "meta": {
    "id": "bo_channel",
    "name": "Channel",
    "version": {"major": 1, "minor": 0, "patch": 0},
    "status": "active"
  },
  "attributes": [
    {"name": "channel_code", "type": "string", "required": true, "description": "WEB, MOBILE, EMAIL, ADVISOR"},
    {"name": "display_name", "type": "string", "required": true},
    {"name": "capabilities", "type": "jsonb", "required": false}
  ],
  "lifecycle": ["ACTIVE", "INACTIVE"]
}');

-- 4. Advisor Augmentation
-- Advisor Queue Item
INSERT INTO meta_objects (id, name, version_major, version_minor, version_patch, status, payload)
VALUES ('bo_advisor_queue_item', 'Advisor Queue Item', 1, 0, 0, 'active', '{
  "meta": {
    "id": "bo_advisor_queue_item",
    "name": "Advisor Queue Item",
    "version": {"major": 1, "minor": 0, "patch": 0},
    "status": "active"
  },
  "attributes": [
    {"name": "priority", "type": "string", "required": true, "default": "MEDIUM"},
    {"name": "due_date", "type": "datetime", "required": false},
    {"name": "status", "type": "string", "required": true, "default": "OPEN"},
    {"name": "assigned_advisor_id", "type": "string", "required": false}
  ],
  "rels": [
    {"name": "client", "target_bo": "bo_client", "cardinality": "one"},
    {"name": "recommendation", "target_bo": "bo_recommendation", "cardinality": "one"}
  ],
  "lifecycle": ["OPEN", "ASSIGNED", "IN_PROGRESS", "RESOLVED", "ESCALATED"]
}');

-- 5. Processes
-- Suitability Review
INSERT INTO meta_processes (id, name, version_major, version_minor, version_patch, status, payload)
VALUES ('bp_suitability_review', 'Suitability Review', 1, 0, 0, 'active', '{
  "meta": {
    "id": "bp_suitability_review",
    "name": "Suitability Review",
    "version": {"major": 1, "minor": 0, "patch": 0},
    "status": "active"
  },
  "states": ["INTAKE", "VALIDATION", "ADVISOR_APPROVAL", "ACTIVE"],
  "transitions": [
    {"from": "INTAKE", "to": "VALIDATION", "action": "submit_form"},
    {"from": "VALIDATION", "to": "ADVISOR_APPROVAL", "guard": "is_valid"},
    {"from": "ADVISOR_APPROVAL", "to": "ACTIVE", "action": "approve"}
  ]
}');

-- Advisor Review (Escalation)
INSERT INTO meta_processes (id, name, version_major, version_minor, version_patch, status, payload)
VALUES ('bp_advisor_review', 'Advisor Review', 1, 0, 0, 'active', '{
  "meta": {
    "id": "bp_advisor_review",
    "name": "Advisor Review",
    "version": {"major": 1, "minor": 0, "patch": 0},
    "status": "active"
  },
  "states": ["QUEUED", "REVIEWING", "APPROVED", "REJECTED"],
  "transitions": [
    {"from": "QUEUED", "to": "REVIEWING", "action": "assign"},
    {"from": "REVIEWING", "to": "APPROVED", "action": "approve"},
    {"from": "REVIEWING", "to": "REJECTED", "action": "reject"}
  ],
  "sla": {"REVIEWING": "PT4H"}
}');

-- Feed Curator
INSERT INTO meta_processes (id, name, version_major, version_minor, version_patch, status, payload)
VALUES ('bp_feed_curator', 'Feed Curator', 1, 0, 0, 'active', '{
  "meta": {
    "id": "bp_feed_curator",
    "name": "Feed Curator",
    "version": {"major": 1, "minor": 0, "patch": 0},
    "status": "active"
  },
  "states": ["GATHERING", "RANKING", "DELIVERING", "COMPLETED"],
  "transitions": [
    {"from": "GATHERING", "to": "RANKING", "action": "compute_scores"},
    {"from": "RANKING", "to": "DELIVERING", "action": "publish_feed"}
  ]
}');

-- 6. Metrics
-- Engagement Index
INSERT INTO meta_metrics (id, name, version_major, version_minor, version_patch, status, payload)
VALUES ('metric_engagement_index', 'Engagement Index', 1, 0, 0, 'active', '{
  "meta": {
    "id": "metric_engagement_index",
    "name": "Engagement Index",
    "version": {"major": 1, "minor": 0, "patch": 0},
    "status": "active"
  },
  "definition": {
    "formula": "(login_count * 0.2) + (feed_clicks * 0.5) + (action_completions * 1.0)",
    "grain": ["client_id", "week"],
    "unit": "SCORE"
  }
}');

-- Advisor Adoption
INSERT INTO meta_metrics (id, name, version_major, version_minor, version_patch, status, payload)
VALUES ('metric_advisor_adoption', 'Advisor Adoption Rate', 1, 0, 0, 'active', '{
  "meta": {
    "id": "metric_advisor_adoption",
    "name": "Advisor Adoption Rate",
    "version": {"major": 1, "minor": 0, "patch": 0},
    "status": "active"
  },
  "definition": {
    "formula": "count(approved_recommendations) / count(total_recommendations)",
    "grain": ["advisor_id", "month"],
    "unit": "PERCENT"
  }
}');
