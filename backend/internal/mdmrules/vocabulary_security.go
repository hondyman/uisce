package mdmrules

// securityVocabulary is the semantic-term vocabulary of the 30 security MDM business objects in the gold-copy
// tenant, as seeded by 20261025_001 from the term mapped to each column (business_object_fields.field_name).
// It is merged into Vocabulary, and the catalog test checks every term a rule references against it.
// Regenerate it from the query in docs/mdm-rules.md if the mappings change.
var securityVocabulary = map[string][]string{
	"business_day_convention":        terms("ID ConventionCode Name Description IsActive CustomAttributes CreatedAt UpdatedAt TenantId"),
	"classification_scheme":          terms("ID SchemeCode Name Version HierarchyDepth IsActive CustomAttributes CreatedAt UpdatedAt TenantId"),
	"classification_scheme_map":      terms("ID FromSchemeCode FromCode FromLevel ToSchemeCode ToCode ToLevel MappingType Confidence IsActive CustomAttributes CreatedAt UpdatedAt TenantId"),
	"day_count_convention":           terms("ID ConventionCode Name Formula IsActive CustomAttributes CreatedAt UpdatedAt TenantId"),
	"payment_frequency":              terms("ID FrequencyCode Name PeriodsPerYear IsActive CustomAttributes CreatedAt UpdatedAt TenantId"),
	"rating_scale":                   terms("ID AgencyCode RatingValue RatingScale NumericEquivalent IsInvestmentGrade IsDefault IsHighYield IsCurrent CustomAttributes CreatedAt UpdatedAt TenantId"),
	"security_asset_class":           terms("ID AssetClassCode Name Description IsActive DisplayOrder CustomAttributes CreatedAt UpdatedAt TenantId"),
	"security_asset_class_routing":   terms("ID SecTypCd SecSubTypeCode DetailTable SatelliteTables RequiresUnderlying RequiresTermSheet RequiresLegalReview IsActive CustomAttributes CreatedAt UpdatedAt TenantId"),
	"security_ca_linkage":            terms("ID SecurityId CorporateActionID LinkageType SourceSystemIdentifier Confidence IsPrimary Status CustomAttributes CreatedAt UpdatedAt TenantId"),
	"security_change_request":        terms("ID RequestReference SecurityId AssetClassCode ChangeType RequestedChanges RequestReason Source Status RequestedBy AssignedTo ApprovedBy ApprovedAt AppliedAt RejectionReason CustomAttributes CreatedAt UpdatedAt TenantId"),
	"security_exception":             terms("ID SecurityId IdentifierValue AssetClassCode ExceptionType Severity ExceptionDescription SourceSystemIdentifier DetectedAt Status AssignedTo ResolvedAt ResolutionNote CustomAttributes TenantId"),
	"security_feed_schedule":         terms("ID SourceSystemIdentifier AssetClassCode FeedName FeedType ExpectedCadence ExpectedDeliveryTime DeliveryTimezone SlaMinutes GracePeriodMinutes IsActive CustomAttributes CreatedAt UpdatedAt TenantId"),
	"security_field_mapping":         terms("ID SourceSystemIdentifier AssetClassCode VendorField VendorDataType InternalTable InternalField TransformExpression IsRequired DefaultValue ValidFrom ValidTo IsActive CustomAttributes CreatedAt UpdatedAt TenantId"),
	"security_field_transform":       terms("ID TransformCode Name TransformType TransformLogic InputDataType OutputDataType IsActive CustomAttributes CreatedAt UpdatedAt TenantId"),
	"security_golden_field":          terms("ID GoldenRecordId FieldName FieldValue FieldValueNumeric FieldValueDate FieldValueJson SourceSystemIdentifier SourceField Confidence RuleApplied TenantId"),
	"security_golden_record":         terms("ID SecurityId GoldenVersion IsCurrent AssetClassCode SecTypCd SecSubTypeCode EffectiveDate KnowledgeTimestamp GoldenAttributes WinningSources OverallDqScore IdentityConfidence Status PublishedAt PublishedBy TenantId"),
	"security_identifier_authority":  terms("ID IdentifierType AssetClassCode SourceSystemIdentifier IsAuthoritative Priority RequiresChecksumValidation ChecksumAlgorithm EffectiveFrom EffectiveTo CustomAttributes CreatedAt UpdatedAt TenantId"),
	"security_identifier_conflict":   terms("ID IdentifierType IdentifierValue SourceSystemIdentifierA SourceValueA SourceSystemIdentifierB SourceValueB SecurityIdA SecurityIdB ConflictType Severity Status ResolutionNote ResolvedBy ResolvedAt CustomAttributes CreatedAt UpdatedAt TenantId"),
	"security_identifier_issuance":   terms("ID SecurityId IdentifierType IdentifierValue IssuingAuthority SourceSystemIdentifier FirstSeenAt LastConfirmedAt IsValid ValidationMethod CustomAttributes CreatedAt TenantId"),
	"security_match_candidate":       terms("ID MatchRuleId SecurityIdA SecurityIdB OverallScore DeterministicMatch MatchedKeys ConflictingKeys Status ReviewedBy ReviewedAt ReviewNote CustomAttributes CreatedAt UpdatedAt TenantId"),
	"security_match_rule":            terms("ID RuleCode AssetClassCode SecSubTypeCode RuleName MatchKeys DeterministicKeys FuzzyKeys ThresholdAutoMatch ThresholdReview ThresholdNoMatch Priority IsActive CustomAttributes CreatedAt UpdatedAt TenantId"),
	"security_reconciliation_result": terms("ID ReconciliationId SecurityId IdentifierValue FieldName ValueA ValueB ValuesMatch VariancePercent Severity Status ResolvedBy ResolvedAt ResolutionNote TenantId"),
	"security_source_priority":       terms("ID AssetClassCode SecSubTypeCode FieldGroup SourceSystemIdentifier Priority IsActive CustomAttributes CreatedAt UpdatedAt TenantId"),
	"security_status_authority":      terms("ID AssetClassCode SecSubTypeCode StatusField SourceSystemIdentifier AuthorityLevel AutomaticApply RequiresReview IsActive CustomAttributes CreatedAt UpdatedAt TenantId"),
	"security_steward":               terms("ID UserId AssetClassCode SecSubTypeCode Region StewardRole CanOverrideSurvivorship CanMerge CanPublish IsActive CustomAttributes CreatedAt UpdatedAt TenantId"),
	"security_survivorship_rule":     terms("ID AssetClassCode SecSubTypeCode FieldGroup FieldName Strategy SourcePriority MinConfidence MaxStalenessHours ManualOverrideAllowed IsActive CustomAttributes CreatedAt UpdatedAt TenantId"),
	"security_term_extraction_field": terms("ID JobIdentifier FieldName ExtractedValue ExtractedValueJson Confidence SourcePage SourceSnippet ValidationStatus ValidationNote ValidatedBy ValidatedAt TenantId"),
	"security_term_sheet":            terms("ID SecurityId DocumentType DocumentName DocumentUrl DocumentHash Language EffectiveDate ExtractionStatus ExtractedAt ExtractedBy ReviewedBy ReviewedAt Confidence CustomAttributes CreatedAt UpdatedAt TenantId"),
	"security_type":                  terms("ID AssetClassCode SecTypCd SecSubTypeCode Name Description DetailTable SatelliteTables RequiresMaturity RequiresCoupon RequiresUnderlying RequiresIssuer IsActive CustomAttributes CreatedAt UpdatedAt TenantId"),
	"security_type_mapping":          terms("ID SourceSystemIdentifier VendorTypeCd VendorSubTypeCd VendorDescription InternalAssetClassCode InternalSecurityTypeCode InternalSecuritySubTypeCode Confidence IsActive CustomAttributes CreatedAt UpdatedAt TenantId"),
}

func init() {
	for bo, ts := range securityVocabulary {
		Vocabulary[bo] = ts
	}
}
