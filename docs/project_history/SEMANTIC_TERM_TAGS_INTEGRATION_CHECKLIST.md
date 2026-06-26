# Semantic Term Tagging System - Integration Checklist

## Pre-Integration (Preparation)

### Documentation Review
- [ ] Read `START_HERE_SEMANTIC_TERM_TAGS.md` (5 minutes)
- [ ] Read `SEMANTIC_TERM_TAGS_QUICK_REFERENCE.md` (10 minutes)
- [ ] Skim `SEMANTIC_TERM_TAGS_INTEGRATION_GUIDE.md` (10 minutes)
- [ ] Review file locations and naming conventions

### Environment Setup
- [ ] Verify PostgreSQL is running and accessible
- [ ] Confirm you have database admin access
- [ ] Check Go compilation tools are available
- [ ] Verify React development environment is ready

---

## Phase 1: Database Migration (1 hour)

### 1.1: Execute Migration
- [ ] Locate: `migrations/add_semantic_term_tags.sql`
- [ ] Run migration:
  ```bash
  psql postgres://user:pass@host:5432/dbname -f migrations/add_semantic_term_tags.sql
  ```
- [ ] Check return code (0 = success)

### 1.2: Verify Migration
- [ ] Connect to database:
  ```bash
  psql postgres://user:pass@host:5432/dbname
  ```
- [ ] Verify tag count:
  ```sql
  SELECT COUNT(*) as tag_count FROM semantic_term_tags;
  -- Expected: 45
  ```
- [ ] Verify categories:
  ```sql
  SELECT DISTINCT tag_category FROM semantic_term_tags ORDER BY tag_category;
  -- Expected: business_area, data_type, domain, governance, sensitivity, usage_pattern
  ```
- [ ] Verify catalog_node column:
  ```sql
  SELECT column_name, data_type FROM information_schema.columns
  WHERE table_name = 'catalog_node' AND column_name = 'tags';
  -- Expected: tags, jsonb
  ```

### 1.3: Backup Database
- [ ] Create backup of migrated database
- [ ] Store backup location: _______________

### Post-Migration Checklist
- [ ] All tables created successfully
- [ ] All indexes created
- [ ] 45 tags inserted
- [ ] JSONB column added to catalog_node
- [ ] Database connection verified

---

## Phase 2: Backend Integration (2-3 hours)

### 2.1: Copy Backend Files
- [ ] Copy `backend/internal/models/semantic_term_tags.go`
- [ ] Copy `backend/internal/services/tag_suggestion_service.go`
- [ ] Copy `backend/internal/api/semantic_term_tags_resolver.go`
- [ ] Verify files are in correct locations

### 2.2: Verify Go Code Compiles
- [ ] Run: `go build ./backend/...`
- [ ] Check for errors: _______________
- [ ] Verify models package compiles:
  ```bash
  cd backend && go build ./internal/models
  ```
- [ ] Verify services package compiles:
  ```bash
  cd backend && go build ./internal/services
  ```

### 2.3: Update GraphQL Schema
- [ ] Locate your main GraphQL schema file
- [ ] Open: `api/semantic_term_tags.graphql`
- [ ] Copy all schema definitions into your main schema
- [ ] Verify schema syntax is valid

### 2.4: Wire GraphQL Resolvers

#### Create Resolver Instance
- [ ] In your GraphQL setup file, create resolver:
  ```go
  tagResolver := api.NewTagResolver(db)
  ```
- [ ] Add to your resolver map:
  ```go
  Query: &QueryResolver{
    TagResolver: tagResolver,
    // ... other resolvers
  }
  ```

#### Implement Query Resolvers
- [ ] Implement: `SemanticTags`
- [ ] Implement: `TagsByCategory`
- [ ] Implement: `SemanticTermTags`
- [ ] Implement: `TagCategories`
- [ ] Implement: `SuggestSemanticTermTags`

#### Implement Mutation Resolvers
- [ ] Implement: `AddTagToSemanticTerm`
- [ ] Implement: `RemoveTagFromSemanticTerm`
- [ ] Implement: `UpdateSemanticTermTags`
- [ ] Implement: `CreateSemanticTag`
- [ ] Implement: `UpdateSemanticTag`
- [ ] Implement: `DeleteSemanticTag`
- [ ] Implement: `AcceptTagSuggestion`
- [ ] Implement: `ApplyTagSuggestions`

### 2.5: Test GraphQL Endpoints
- [ ] Start GraphQL server
- [ ] Open GraphQL Playground
- [ ] Test query: `{ semanticTags { tagKey tagLabel tagCategory } }`
- [ ] Result: 45 tags returned ✓
- [ ] Test query: `{ tagCategories { category tags { tagLabel } } }`
- [ ] Result: 6 categories returned ✓
- [ ] Test mutation: `applyTagSuggestions(...)`
- [ ] Result: Tags applied successfully ✓

### Backend Integration Complete
- [ ] Go code compiles without errors
- [ ] GraphQL schema merged
- [ ] Resolvers wired and registered
- [ ] All GraphQL operations tested
- [ ] Database queries working

---

## Phase 3: Frontend Integration (2-3 hours)

### 3.1: Copy Frontend Files
- [ ] Copy `frontend/src/components/SemanticTermTags/SemanticTermTags.tsx`
- [ ] Copy `frontend/src/components/SemanticTermTags/SemanticTermTags.css`
- [ ] Verify files in correct location

### 3.2: Update Import Paths
- [ ] Open `SemanticTermTags.tsx`
- [ ] Update any import paths to match your project structure
- [ ] Update GraphQL client imports if needed
- [ ] Update relative imports for CSS

### 3.3: Integrate into Semantic Term Form

#### Add Tag Editor Component
- [ ] Locate your semantic term create/edit form
- [ ] Import component:
  ```tsx
  import { SemanticTermTagsEditor } from './components/SemanticTermTags/SemanticTermTags';
  import './components/SemanticTermTags/SemanticTermTags.css';
  ```
- [ ] Add state for tags:
  ```tsx
  const [tags, setTags] = useState<string[]>([]);
  ```
- [ ] Add to JSX:
  ```tsx
  <SemanticTermTagsEditor
    termId={existingTerm?.id}
    currentTags={tags}
    onTagsChange={setTags}
    readOnly={false}
  />
  ```

#### Add Tag Suggestion Wizard
- [ ] Import component:
  ```tsx
  import { TagSuggestionWizard } from './components/SemanticTermTags/SemanticTermTags';
  ```
- [ ] Add state for wizard visibility:
  ```tsx
  const [showWizard, setShowWizard] = useState(false);
  ```
- [ ] Add button:
  ```tsx
  <button onClick={() => setShowWizard(true)}>Get Tag Suggestions</button>
  ```
- [ ] Add wizard component:
  ```tsx
  {showWizard && (
    <TagSuggestionWizard
      termName={formData.nodeName}
      displayName={formData.displayName}
      description={formData.description}
      dataType={formData.dataType}
      domain={formData.domain}
      expression={formData.expression}
      existingTags={tags}
      onApplySuggestions={(suggested) => {
        setTags([...tags, ...suggested]);
        setShowWizard(false);
      }}
      onCancel={() => setShowWizard(false)}
    />
  )}
  ```

### 3.4: Test React Components

#### Component Rendering
- [ ] Start React development server
- [ ] Navigate to semantic term form
- [ ] Verify tag editor renders
- [ ] Verify CSS loads (check styles applied)
- [ ] Verify wizard button appears

#### User Interactions
- [ ] Type in tag search field
- [ ] Verify dropdown opens
- [ ] Select a tag
- [ ] Verify tag appears as pill
- [ ] Click remove button
- [ ] Verify tag removed

#### Wizard Functionality
- [ ] Click "Get Suggestions"
- [ ] Verify wizard modal opens
- [ ] Verify suggestions load (GraphQL call)
- [ ] Verify confidence bars display
- [ ] Select some suggestions
- [ ] Click "Apply Selected Tags"
- [ ] Verify tags applied to editor
- [ ] Verify wizard closes

### Frontend Integration Complete
- [ ] Components compile without errors
- [ ] CSS loads and styles apply
- [ ] Tag editor renders in form
- [ ] Tag selection works
- [ ] Wizard opens and suggests tags
- [ ] Tags persist after selection
- [ ] UI is responsive

---

## Phase 4: End-to-End Testing (1-2 hours)

### 4.1: Create New Semantic Term

- [ ] Open semantic term creation form
- [ ] Fill in basic information:
  - [ ] Node Name: `test_revenue`
  - [ ] Display Name: `Test Revenue`
  - [ ] Data Type: `NUMERIC`
  - [ ] Domain: `sales`
  - [ ] Description: `Test revenue metric`

### 4.2: Test Tag Suggestions

- [ ] Click "Get Suggestions" button
- [ ] Verify wizard opens
- [ ] Verify suggestions appear (should show: numeric, measure, sales)
- [ ] Verify confidence bars display
- [ ] Verify pre-selected items (>0.8 confidence)
- [ ] Deselect one suggestion
- [ ] Click "Apply Selected Tags"
- [ ] Verify selected tags appear in editor

### 4.3: Test Manual Tag Addition

- [ ] Type in tag search field: "fin"
- [ ] Verify dropdown filters tags
- [ ] Select "finance" tag
- [ ] Verify tag appears in editor
- [ ] Test removing tag
- [ ] Verify tag removed

### 4.4: Save and Verify

- [ ] Click "Save Semantic Term"
- [ ] Verify save succeeds
- [ ] Verify GraphQL mutation succeeded
- [ ] Check database: `SELECT tags FROM catalog_node WHERE id = 'test_...';`
- [ ] Verify tags stored as JSONB

### 4.5: Test Multi-term Operations

- [ ] Create 2-3 more semantic terms
- [ ] Apply tags to each
- [ ] Verify tags displayed in list view
- [ ] Test batch operations (if available)

### 4.6: Performance Testing

- [ ] Create 20+ semantic terms
- [ ] Verify tag suggestions still fast (<2 seconds)
- [ ] Verify tag editor responsive
- [ ] Check database query performance

### End-to-End Testing Complete
- [ ] Create semantic term: ✓
- [ ] Get tag suggestions: ✓
- [ ] Apply suggestions: ✓
- [ ] Manual tag addition: ✓
- [ ] Save and verify: ✓
- [ ] Tags persist: ✓
- [ ] Performance acceptable: ✓

---

## Phase 5: Validation & Troubleshooting

### 5.1: Database Validation
- [ ] Verify 45 tags in semantic_term_tags:
  ```sql
  SELECT COUNT(*) FROM semantic_term_tags WHERE is_active = true;
  ```
- [ ] Verify tag categories:
  ```sql
  SELECT DISTINCT tag_category FROM semantic_term_tags;
  ```
- [ ] Check semantic_term_tag_suggestions table is empty initially:
  ```sql
  SELECT COUNT(*) FROM semantic_term_tag_suggestions;
  ```

### 5.2: GraphQL Validation
- [ ] Test all 5 queries
- [ ] Test all 8 mutations
- [ ] Verify response times < 500ms
- [ ] Check error handling with invalid inputs

### 5.3: Frontend Validation
- [ ] Test on Chrome, Firefox, Safari
- [ ] Test on mobile viewport
- [ ] Verify keyboard navigation works
- [ ] Test accessibility with screen reader (if possible)

### 5.4: Troubleshooting

#### If tags not appearing:
- [ ] Check database migration executed
- [ ] Verify GraphQL queries return data
- [ ] Check browser console for errors
- [ ] Verify CSS imports correct path

#### If suggestions not accurate:
- [ ] Check term fields populated correctly
- [ ] Review inference strategy (6 methods)
- [ ] Verify confidence scores calculated
- [ ] Check tag_suggestion_service logic

#### If performance issues:
- [ ] Check database indexes exist
- [ ] Monitor query execution times
- [ ] Consider pagination for large result sets
- [ ] Check for N+1 query problems

### Validation Complete
- [ ] Database correct
- [ ] GraphQL working
- [ ] Frontend rendering
- [ ] No errors in logs
- [ ] Performance acceptable

---

## Phase 6: Production Deployment (0.5-1 hour)

### 6.1: Pre-Deployment Checklist
- [ ] All phases complete and tested
- [ ] Code reviewed by team
- [ ] Database backed up
- [ ] Deployment plan approved
- [ ] Rollback plan documented

### 6.2: Deploy to Staging
- [ ] Deploy database migration
- [ ] Deploy backend code
- [ ] Deploy frontend code
- [ ] Run smoke tests
- [ ] Verify all functionality works

### 6.3: Deploy to Production
- [ ] Execute database migration
- [ ] Deploy backend service
- [ ] Deploy frontend application
- [ ] Monitor logs for errors
- [ ] Verify all endpoints responding

### 6.4: Post-Deployment
- [ ] Verify all features working
- [ ] Check performance metrics
- [ ] Monitor error rates
- [ ] Get user feedback
- [ ] Document any issues

### Production Deployment Complete
- [ ] Staging deployment: ✓
- [ ] Production deployment: ✓
- [ ] All tests passing: ✓
- [ ] User feedback positive: ✓
- [ ] No critical issues: ✓

---

## Summary Checklist

### Must Complete Before Going Live
- [ ] All 6 phases complete
- [ ] End-to-end testing passed
- [ ] No critical errors
- [ ] Database migration verified
- [ ] Performance acceptable
- [ ] Documentation reviewed

### Status: ___________________________
- Date Started: _______________
- Date Completed: _______________
- Total Hours: _______________
- Issues Encountered: _______________
- Lessons Learned: _______________

---

## Success Criteria

✅ **Complete** when you can:

1. Create a semantic term with tags
2. See intelligent tag suggestions
3. Apply suggestions with one click
4. See tags persisted in database
5. View tags in list view
6. Perform batch tag operations
7. All performance metrics acceptable
8. No errors in logs

---

## Support Resources

| Situation | Solution |
|-----------|----------|
| Syntax errors in Go code | Review `semantic_term_tags.go` for imports |
| GraphQL schema errors | Merge schema carefully, check syntax |
| React component not rendering | Verify CSS imported, check browser console |
| Tags not appearing | Check database migration executed |
| Slow performance | Check database indexes, review queries |
| GraphQL timeouts | Increase timeout limits, optimize queries |

---

## Final Notes

- **Total Estimated Time**: 6-9 hours
- **Break into**: 1.5 + 2.5 + 2.5 + 1.5 hours per phase
- **Best Practice**: Test each phase before moving to next
- **Documentation**: Keep this checklist handy during integration
- **Questions**: Refer to docs for answers before seeking support

---

**Good luck with your integration! 🚀**

Last updated: January 4, 2025
