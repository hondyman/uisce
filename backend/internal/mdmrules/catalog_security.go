package mdmrules

import "fmt"

// Two shapes the security rules need that the earlier tiers did not.

// nonNegative: a nullable numeric term, when present, must be >= 0 (zero is meaningful, unlike positive).
func nonNegative(bo, term string) Rule {
	return Rule{BO: bo, Name: fmt.Sprintf("mdm.%s.%s_non_negative", bo, snake(term)),
		Description: term + " must not be negative when set.",
		Severity:    warnSev, Timing: "pre_write", Category: catIntegrity,
		AST: group("OR", isNull(term), cond(term, ">=", float64(0))),
		Cases: []Case{
			{"null allowed", map[string]any{term: nil}, true},
			{"zero", map[string]any{term: float64(0)}, true},
			{"positive", map[string]any{term: float64(15)}, true},
			{"negative", map[string]any{term: float64(-1)}, false},
		}}
}

// notBoth: two flags that exclude each other must not both be true.
func notBoth(bo, name, description, a, b string) Rule {
	return Rule{BO: bo, Name: fmt.Sprintf("mdm.%s.%s", bo, name), Description: description,
		Severity: blockSev, Timing: "pre_write", Category: catIntegrity,
		AST: group("OR", cond(a, "==", false), cond(b, "==", false)),
		Cases: []Case{
			{"neither", map[string]any{a: false, b: false}, true},
			{a + " only", map[string]any{a: true, b: false}, true},
			{b + " only", map[string]any{a: false, b: true}, true},
			{"both", map[string]any{a: true, b: true}, false},
		}}
}

// securityRules is the rule set for the 30 security MDM business objects (tables created by
// backend/db/crims/0001_mdm_security.sql, BOs by migration 20261025_001). Same guard rails as the earlier
// tiers: rules are on semantic terms; nothing on Status, Priority or an IsActive flag (enumerations and UX
// controls own those); only checks the engine can evaluate today (see docs/mdm-rules.md, "Engine gaps").
func securityRules() []Rule {
	var r []Rule
	add := func(x ...Rule) { r = append(r, x...) }

	// ---- required terms: what makes a record identifiable and usable --------------------------------------
	add(
		// vocabularies and classification
		required("security_asset_class", "An asset class needs a code and a name.", ScopeAll, catIntegrity, "AssetClassCode", "Name"),
		required("security_type", "A security type needs its asset class, type, sub-type and name.", ScopeAll, catIntegrity,
			"AssetClassCode", "SecTypCd", "SecSubTypeCode", "Name"),
		required("security_type_mapping", "A type mapping needs the source, the vendor type and the internal asset class, type and sub-type.", ScopeAll, catIntegrity,
			"SourceSystemIdentifier", "VendorTypeCd", "InternalAssetClassCode", "InternalSecurityTypeCode", "InternalSecuritySubTypeCode"),
		required("classification_scheme", "A classification scheme needs a code and a name.", ScopeAll, catIntegrity, "SchemeCode", "Name"),
		required("classification_scheme_map", "A scheme mapping needs both schemes, both codes and the mapping type.", ScopeAll, catIntegrity,
			"FromSchemeCode", "FromCode", "ToSchemeCode", "ToCode", "MappingType"),
		required("rating_scale", "A rating needs the agency, the value, the scale and its numeric equivalent.", ScopeAll, catIntegrity,
			"AgencyCode", "RatingValue", "RatingScale", "NumericEquivalent"),
		required("day_count_convention", "A day-count convention needs a code and a name.", ScopeAll, catIntegrity, "ConventionCode", "Name"),
		required("business_day_convention", "A business-day convention needs a code and a name.", ScopeAll, catIntegrity, "ConventionCode", "Name"),
		required("payment_frequency", "A payment frequency needs a code, a name and the periods per year.", ScopeAll, catIntegrity,
			"FrequencyCode", "Name", "PeriodsPerYear"),
		// source management and mapping
		required("security_source_priority", "A source priority needs the asset class, the field group and the source.", ScopeAll, catIntegrity,
			"AssetClassCode", "FieldGroup", "SourceSystemIdentifier"),
		required("security_feed_schedule", "A feed schedule needs the source, the feed name and the feed type.", ScopeAll, catIntegrity,
			"SourceSystemIdentifier", "FeedName", "FeedType"),
		required("security_asset_class_routing", "A routing entry needs the type, the sub-type and the detail table.", ScopeAll, catIntegrity,
			"SecTypCd", "SecSubTypeCode", "DetailTable"),
		required("security_identifier_authority", "An identifier authority needs the identifier type and the source.", ScopeAll, catIntegrity,
			"IdentifierType", "SourceSystemIdentifier"),
		required("security_field_mapping", "A field mapping needs the source, the vendor field and the internal table and field.", ScopeAll, catIntegrity,
			"SourceSystemIdentifier", "VendorField", "InternalTable", "InternalField"),
		required("security_field_transform", "A transform needs a code, a name, a type and its logic.", ScopeAll, catIntegrity,
			"TransformCode", "Name", "TransformType", "TransformLogic"),
		required("security_survivorship_rule", "A survivorship rule needs the asset class, field group, field and strategy.", ScopeAll, catIntegrity,
			"AssetClassCode", "FieldGroup", "FieldName", "Strategy"),
		required("security_match_rule", "A match rule needs a code, the asset class, a name and its match keys.", ScopeAll, catIntegrity,
			"RuleCode", "AssetClassCode", "RuleName", "MatchKeys"),
		required("security_status_authority", "A status authority needs the asset class, the status field, the source and the authority level.", ScopeAll, catIntegrity,
			"AssetClassCode", "StatusField", "SourceSystemIdentifier", "AuthorityLevel"),
		// identifiers, matching and golden record
		required("security_identifier_issuance", "An issued identifier needs the security, the type, the value and the source.", ScopeAll, catIntegrity,
			"SecurityIdentifier", "IdentifierType", "IdentifierValue", "SourceSystemIdentifier"),
		required("security_identifier_conflict", "An identifier conflict needs the identifier, both sources, the conflict type and a severity.", ScopeAll, catIntegrity,
			"IdentifierType", "IdentifierValue", "SourceSystemIdentifierA", "SourceSystemIdentifierB", "ConflictType", "Severity"),
		required("security_match_candidate", "A match candidate needs the rule, both securities and the score.", ScopeAll, catIntegrity,
			"MatchRuleId", "SecurityIdentifierA", "SecurityIdentifierB", "OverallScore"),
		required("security_golden_record", "A golden record needs the security, its classification, the effective date and its attributes and winning sources.", ScopeAll, catIntegrity,
			"SecurityIdentifier", "AssetClassCode", "SecTypCd", "SecSubTypeCode", "EffectiveDate", "GoldenAttributes", "WinningSources"),
		required("security_golden_field", "A golden field needs its golden record and the field name.", ScopeAll, catIntegrity, "GoldenRecordId", "FieldName"),
		// documents, reconciliation and governance
		required("security_term_sheet", "A term sheet needs the security, the document type and the document name.", ScopeAll, catIntegrity,
			"SecurityIdentifier", "DocumentType", "DocumentName"),
		required("security_term_extraction_field", "An extracted value needs its job and the field name.", ScopeAll, catIntegrity, "JobIdentifier", "FieldName"),
		required("security_reconciliation_result", "A reconciliation result needs its reconciliation, the field and a severity.", ScopeAll, catIntegrity,
			"ReconciliationId", "FieldName", "Severity"),
		required("security_exception", "A security exception needs a type, a severity and a description.", ScopeAll, catIntegrity,
			"ExceptionType", "Severity", "ExceptionDescription"),
		required("security_steward", "A security steward needs a user and a role.", ScopeAll, catIntegrity, "UserId", "StewardRole"),
		required("security_change_request", "A change request needs a reference, a type, the requested changes and the requester.", ScopeAll, catIntegrity,
			"RequestReference", "ChangeType", "RequestedChanges", "RequestedBy"),
		required("security_ca_linkage", "A corporate-action link needs the security, the corporate action and the link type.", ScopeAll, catIntegrity,
			"SecurityIdentifier", "CorporateActionID", "LinkageType"),
	)

	// ---- numeric: scores and confidences are 0-100 (numeric(5,2)); counts and durations are positive ------
	add(
		atLeast("classification_scheme", "HierarchyDepth", 1),
		atLeast("classification_scheme_map", "FromLevel", 1),
		atLeast("classification_scheme_map", "ToLevel", 1),
		inRange("classification_scheme_map", "Confidence", 0, 100),
		positive("payment_frequency", "PeriodsPerYear"),
		positive("security_feed_schedule", "SlaMinutes"),
		nonNegative("security_feed_schedule", "GracePeriodMinutes"),
		inRange("security_type_mapping", "Confidence", 0, 100),
		inRange("security_survivorship_rule", "MinConfidence", 0, 100),
		positive("security_survivorship_rule", "MaxStalenessHours"),
		chain3("security_match_rule", "thresholds_ordered", "AutoMatch threshold must be >= Review threshold >= NoMatch threshold.",
			"ThresholdAutoMatch", "ThresholdReview", "ThresholdNoMatch"),
		inRange("security_match_rule", "ThresholdAutoMatch", 0, 100),
		inRange("security_match_rule", "ThresholdReview", 0, 100),
		inRange("security_match_rule", "ThresholdNoMatch", 0, 100),
		inRange("security_match_candidate", "OverallScore", 0, 100),
		atLeast("security_golden_record", "GoldenVersion", 1),
		inRange("security_golden_record", "OverallDqScore", 0, 100),
		inRange("security_golden_record", "IdentityConfidence", 0, 100),
		inRange("security_golden_field", "Confidence", 0, 100),
		inRange("security_term_sheet", "Confidence", 0, 100),
		inRange("security_term_extraction_field", "Confidence", 0, 100),
		positive("security_term_extraction_field", "SourcePage"),
		inRange("security_ca_linkage", "Confidence", 0, 100),
	)

	// ---- consistency: things that must be recorded together, and flags that exclude each other -----------
	add(
		notBoth("rating_scale", "not_investment_grade_and_high_yield",
			"A rating cannot be both investment grade and high yield.", "IsInvestmentGrade", "IsHighYield"),
		requiredIf("security_identifier_authority", "checksum_algorithm_when_required",
			"When an identifier requires checksum validation, the checksum algorithm must be named.", "RequiresChecksumValidation", "ChecksumAlgorithm"),
		bothOrNeither("security_golden_record", "publication_recorded_completely",
			"PublishedAt and PublishedBy must be recorded together: a half-published golden record is inconsistent.", "PublishedAt", "PublishedBy"),
		bothOrNeither("security_match_candidate", "review_recorded_completely",
			"ReviewedBy and ReviewedAt must be recorded together.", "ReviewedBy", "ReviewedAt"),
		bothOrNeither("security_identifier_conflict", "resolution_recorded_completely",
			"ResolvedBy and ResolvedAt must be recorded together.", "ResolvedBy", "ResolvedAt"),
		bothOrNeither("security_reconciliation_result", "resolution_recorded_completely",
			"ResolvedBy and ResolvedAt must be recorded together.", "ResolvedBy", "ResolvedAt"),
		bothOrNeither("security_term_extraction_field", "validation_recorded_completely",
			"ValidatedBy and ValidatedAt must be recorded together.", "ValidatedBy", "ValidatedAt"),
		bothOrNeither("security_term_sheet", "extraction_recorded_completely",
			"ExtractedBy and ExtractedAt must be recorded together.", "ExtractedBy", "ExtractedAt"),
		bothOrNeither("security_term_sheet", "review_recorded_completely",
			"ReviewedBy and ReviewedAt must be recorded together.", "ReviewedBy", "ReviewedAt"),
	)
	return r
}
