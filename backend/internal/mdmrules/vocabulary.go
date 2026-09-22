package mdmrules

import "strings"

// Vocabulary is the semantic-term vocabulary of each tier 1/2 MDM BO in the gold-copy tenant, as
// seeded by 20261022_001 / 20261023_001..003 from the terms mapped to each column (business_object_fields
// .field_name). Rules reference these terms, never physical columns. The catalog test checks every
// term a rule references against this list so a typo fails offline instead of becoming a rule_error at
// write time. Regenerate it from the query in docs/mdm-rules.md if the mappings change.
//
// attribute_def is absent on purpose: it has no mapped terms, so no rule can reference it yet.
var Vocabulary = map[string][]string{
	"benchmark":                   terms("BenchmarkId BenchmarkCode BenchmarkName BenchmarkType Currency RebalanceFrequency Source BenchmarkIsActive CustomAttributes CreatedAt UpdatedAt TenantId BasketSecurityId"),
	"change_request":              terms("ChangeId EntityId EntityType ChangeType ProposedChanges Status SubmittedBy ReviewedBy ReviewedAt ReviewNotes CustomAttributes CreatedAt UpdatedAt TenantId"),
	"dq_issue":                    terms("DqId EntityId EntityType RuleId IssueDescription Severity Status AssignedTo ResolvedAt ResolutionNotes CustomAttributes CreatedAt UpdatedAt TenantId"),
	"dq_rule":                     terms("DqId RuleName EntityType CheckExpression Severity DqDescription DqIsActive CustomAttributes CreatedAt UpdatedAt TenantId"),
	"hierarchy":                   terms("HierarchyId HierarchyType HierarchyName RootEntityType RootEntityId HierarchyDescription HierarchyIsActive CustomAttributes CreatedAt UpdatedAt TenantId"),
	"issuer":                      terms("IssuerId IssuerCode IssuerName CountryCode CustomAttributes CreatedAt UpdatedAt TenantId SourceSystemIdentifier GoldenRecordId DqScore LegalName ShortName DisplayName IssuerType IssuerSubType Lei LegalForm Domicile Jurisdiction CountryOfRisk CountryOfIncorporation IncorporationDate DissolutionDate RegistrationNumber RegistrationAuthority TaxId UltimateParentId ImmediateParentIdentifier IssuerGroupId Status StatusReason IsFinancial IsRegulated IsSovereign IsGse IsSpv IsShell IsGreenIssuer IsSustainableIssuer PrimarySectorCode PrimaryIndustryCode EmployeeCount MarketCapUsd InactivatedAt"),
	"issuer_change_request":       terms("IssuerId RequestReference IssuerType ChangeType RequestedChanges RequestReason Source Status RequestedBy AssignedTo ApprovedBy ApprovedAt AppliedAt RejectionReason CustomAttributes CreatedAt UpdatedAt TenantId"),
	"issuer_exception":            terms("IssuerId IdentifierValue IssuerType ExceptionType Severity ExceptionDescription SourceSystemIdentifier DetectedAt Status AssignedTo ResolvedAt ResolutionNote CustomAttributes TenantId"),
	"issuer_field_mapping":        terms("IssuerId SourceSystemIdentifier IssuerType VendorField InternalTable InternalField TransformExpression IsRequired DefaultValue ValidFrom ValidTo IssuerIsActive CustomAttributes CreatedAt UpdatedAt TenantId"),
	"issuer_golden_record":        terms("IssuerId GoldenVersion IsCurrent IssuerType IssuerSubType EffectiveDate KnowledgeTimestamp GoldenAttributes WinningSources OverallDqScore IdentityConfidence HierarchyConfidence Status PublishedAt PublishedBy TenantId"),
	"issuer_hierarchy_review":     terms("IssuerId ParentIssuerId ChildIssuerId HierarchyType ProposedChange ChangeType Source Confidence Status ReviewedBy ReviewedAt ReviewNote CustomAttributes CreatedAt UpdatedAt TenantId"),
	"issuer_hierarchy_rule":       terms("IssuerId HierarchyType RuleCode IssuerName RuleExpression Severity IsBlocking IssuerIsActive CustomAttributes CreatedAt UpdatedAt TenantId"),
	"issuer_identifier_authority": terms("IssuerId IdType IssuerType SourceSystemIdentifier IsAuthoritative Priority RequiresChecksum ChecksumAlgorithm EffectiveFrom EffectiveTo CustomAttributes CreatedAt UpdatedAt TenantId"),
	"issuer_match_candidate":      terms("IssuerId MatchRuleId IssuerIdA IssuerIdB OverallScore DeterministicMatch MatchedKeys ConflictingKeys Status ReviewedBy ReviewedAt ReviewNote CustomAttributes CreatedAt UpdatedAt TenantId"),
	"issuer_match_rule":           terms("IssuerId RuleCode IssuerType IssuerSubType RuleName MatchKeys DeterministicKeys FuzzyKeys ThresholdAutoMatch ThresholdReview ThresholdNoMatch Priority IssuerIsActive CustomAttributes CreatedAt UpdatedAt TenantId"),
	"issuer_steward":              terms("IssuerId UserId IssuerType IssuerSubType Region StewardRole CanOverrideSurvivorship CanMerge CanEditHierarchy CanPublish IssuerIsActive CustomAttributes CreatedAt UpdatedAt TenantId"),
	"issuer_survivorship_rule":    terms("IssuerId IssuerType IssuerSubType FieldGroup FieldName Strategy SourcePriority MinConfidence MaxStalenessHours ManualOverrideAllowed IssuerIsActive CustomAttributes CreatedAt UpdatedAt TenantId"),
	"issuer_type_mapping":         terms("IssuerId SourceSystemIdentifier VendorTypeCd VendorSubTypeCd InternalIssuerType InternalIssuerSubType IsFinancial IsSovereign IsSpv Confidence IssuerIsActive CustomAttributes CreatedAt UpdatedAt TenantId"),
	"mandate":                     terms("MandateId MandateCode PortfolioId StrategyType BenchmarkId TargetReturn RiskTolerance InceptionDate MandateDescription Status CustomAttributes CreatedAt UpdatedAt TenantId"),
	"match_candidate":             terms("MatchId MatchRuleId EntityId1 EntityId2 Score Status ReviewedBy ReviewedAt ReviewNotes CustomAttributes CreatedAt UpdatedAt TenantId"),
	"match_rule":                  terms("MatchId RuleName EntityType MatchFields MatchAlgorithm Threshold AutoMergeThreshold ReviewThreshold MatchIsActive CustomAttributes CreatedAt UpdatedAt TenantId"),
	"party":                       terms("PartyId PartyCode LegalName PartyType Segment TaxId Domicile CustomAttributes CreatedAt UpdatedAt TenantId"),
	"portfolio":                   terms("PortfolioId PortfolioCode PortfolioName PortfolioType ManagerID PortfolioDescription Status CustomAttributes CreatedAt UpdatedAt TenantId"),
	"portfolio_composite":         terms("PortfolioId CompositeCode PortfolioName PortfolioDescription Status CustomAttributes CreatedAt UpdatedAt TenantId"),
	"security":                    terms("ID SecId SecName SecTypCd Isin Cusip Ticker IssuerId AssetCrrncyCd CustomAttributes CreatedAt UpdatedAt"),
	"source_system":               terms("SourceId SourceCode SourceName SystemType Vendor ConnectionInfo Priority SourceIsActive CustomAttributes CreatedAt UpdatedAt TenantId"),
	"steward":                     terms("StewardId StewardName StewardEmail Team Role StewardIsActive CustomAttributes CreatedAt UpdatedAt TenantId"),
	"survivorship_rule":           terms("SurvivorshipId EntityType AttributeName Strategy PriorityVendors AnomalyTolerancePercent StalenessMaxAgeSec SurvivorshipIsActive CustomAttributes CreatedAt UpdatedAt TenantId"),
}

func terms(s string) []string { return strings.Fields(s) }
