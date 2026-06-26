# Semantic Term Tagging System - Integration Guide

## Overview

This guide explains how to integrate the semantic term tagging system into your existing codebase. The system consists of:

1. **Database Layer**: PostgreSQL schema for tags storage and suggestion tracking
2. **Service Layer**: Tag suggestion engine with 6 inference strategies
3. **API Layer**: GraphQL resolvers and HTTP handlers
4. **UI Layer**: React components for tag management and wizard

## Step 1: Execute Database Migration

Before running the system, execute the migration to create necessary tables:

```bash
# Using psql
psql postgres://postgres:postgres@host.docker.internal:5432/alpha?sslmode=disable \
  -f migrations/add_semantic_term_tags.sql

# Or using Go migration tool if available
go run ./cmd/migrate -file migrations/add_semantic_term_tags.sql
```

### Created Tables

- **semantic_term_tags**: Predefined tag definitions (45 tags across 6 categories)
- **semantic_term_tag_suggestions**: Tracking of suggestion acceptance/rejection
- **catalog_node.tags**: JSONB column for storing tags on any catalog node

## Step 2: Wire GraphQL Resolvers

Update your GraphQL server setup to include the tag resolvers:

```go
// In your main GraphQL setup file (e.g., api/graphql.go)

import (
    "semlayer/backend/internal/api"
)

func SetupGraphQL(db *sql.DB) *graphql.Schema {
    // ... existing setup ...
    
    // Create tag resolver
    tagResolver := api.NewTagResolver(db)
    
    // Add to resolver map
    resolvers := &Resolvers{
        Query: &QueryResolver{
            TagResolver: tagResolver,
        },
        Mutation: &MutationResolver{
            TagResolver: tagResolver,
        },
    }
    
    // ... continue with schema creation ...
}
```

### Required Query Resolvers

```go
// In your QueryResolver struct
type QueryResolver struct {
    TagResolver *api.TagResolver
}

// Implement GraphQL query handlers
func (r *QueryResolver) SemanticTags(ctx context.Context) ([]*models.Tag, error) {
    return r.TagResolver.SemanticTags(ctx)
}

func (r *QueryResolver) TagsByCategory(ctx context.Context, args struct{ Category string }) ([]*models.Tag, error) {
    return r.TagResolver.TagsByCategory(ctx, args.Category)
}

func (r *QueryResolver) SemanticTermTags(ctx context.Context, args struct{ TermID string }) ([]*models.Tag, error) {
    return r.TagResolver.SemanticTermTags(ctx, args.TermID)
}

func (r *QueryResolver) TagCategories(ctx context.Context) ([]*models.TagCategory, error) {
    return r.TagResolver.TagCategories(ctx)
}

func (r *QueryResolver) SuggestSemanticTermTags(ctx context.Context, args struct{ Input *models.TagSuggestionRequest }) (*models.TagSuggestionResponse, error) {
    return r.TagResolver.SuggestSemanticTermTags(ctx, args.Input)
}
```

### Required Mutation Resolvers

```go
// In your MutationResolver struct
type MutationResolver struct {
    TagResolver *api.TagResolver
}

// Implement GraphQL mutation handlers
func (r *MutationResolver) AddTagToSemanticTerm(ctx context.Context, args struct {
    Input struct {
        TermID string
        TagKey string
    }
}) error {
    return r.TagResolver.AddTagToSemanticTerm(ctx, args.Input.TermID, args.Input.TagKey)
}

func (r *MutationResolver) RemoveTagFromSemanticTerm(ctx context.Context, args struct {
    TermID string
    TagKey string
}) error {
    return r.TagResolver.RemoveTagFromSemanticTerm(ctx, args.TermID, args.TagKey)
}

func (r *MutationResolver) UpdateSemanticTermTags(ctx context.Context, args struct {
    TermID string
    TagKeys []string
}) error {
    return r.TagResolver.UpdateSemanticTermTags(ctx, args.TermID, args.TagKeys)
}

func (r *MutationResolver) CreateSemanticTag(ctx context.Context, args struct {
    Input *models.TagInput
}) (*models.Tag, error) {
    return r.TagResolver.CreateSemanticTag(ctx, args.Input)
}

func (r *MutationResolver) UpdateSemanticTag(ctx context.Context, args struct {
    TagKey string
    Input  *models.TagInput
}) (*models.Tag, error) {
    return r.TagResolver.UpdateSemanticTag(ctx, args.TagKey, args.Input)
}

func (r *MutationResolver) DeleteSemanticTag(ctx context.Context, args struct {
    TagKey string
}) (bool, error) {
    return r.TagResolver.DeleteSemanticTag(ctx, args.TagKey)
}

func (r *MutationResolver) AcceptTagSuggestion(ctx context.Context, args struct {
    TermID    string
    TagKey    string
    IsAccepted bool
}) error {
    return r.TagResolver.AcceptTagSuggestion(ctx, args.TermID, args.TagKey, args.IsAccepted)
}

func (r *MutationResolver) ApplyTagSuggestions(ctx context.Context, args struct {
    TermID         string
    SuggestedTags  []string
}) error {
    return r.TagResolver.ApplyTagSuggestions(ctx, args.TermID, args.SuggestedTags)
}
```

## Step 3: Integrate React Components

### Import Components in Semantic Term Forms

```tsx
// In your semantic term create/edit modal (e.g., SemanticTermForm.tsx)

import { SemanticTermTagsEditor } from './components/SemanticTermTags/SemanticTermTags';
import { TagSuggestionWizard } from './components/SemanticTermTags/SemanticTermTags';
import './components/SemanticTermTags/SemanticTermTags.css';

export function SemanticTermForm({ existingTerm, onSave }) {
    const [tags, setTags] = useState<string[]>([]);
    const [suggestTags, setSuggestTags] = useState(false);
    
    // ... other form state ...
    
    return (
        <form onSubmit={handleSubmit}>
            {/* ... other fields ... */}
            
            {/* Tag Editor */}
            <SemanticTermTagsEditor
                termId={existingTerm?.id}
                currentTags={tags}
                onTagsChange={setTags}
                readOnly={false}
            />
            
            {/* Tag Suggestion Wizard (shown on new term creation) */}
            {!existingTerm && suggestTags && (
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
                        setSuggestTags(false);
                    }}
                    onCancel={() => setSuggestTags(false)}
                />
            )}
            
            <button 
                type="button" 
                onClick={() => setSuggestTags(true)}
                className="btn btn-secondary"
            >
                Suggest Tags
            </button>
            
            <button type="submit">Save Semantic Term</button>
        </form>
    );
}
```

### Add to Semantic Term List

```tsx
// In semantic term list/table component

import { TagStatistics } from './components/SemanticTermTags/SemanticTermTags';

export function SemanticTermList() {
    return (
        <div>
            <h2>Semantic Terms</h2>
            
            {/* Statistics section */}
            <TagStatistics termId={selectedTermId} />
            
            {/* Term table with tag display */}
            <table>
                <thead>
                    <tr>
                        <th>Name</th>
                        <th>Type</th>
                        <th>Tags</th>
                    </tr>
                </thead>
                <tbody>
                    {terms.map(term => (
                        <tr key={term.id}>
                            <td>{term.displayName}</td>
                            <td>{term.dataType}</td>
                            <td>
                                <div style={{ display: 'flex', flexWrap: 'wrap', gap: '4px' }}>
                                    {term.tags?.map(tag => (
                                        <span 
                                            key={tag.tag_key}
                                            style={{
                                                padding: '4px 8px',
                                                borderRadius: '12px',
                                                background: tag.color_code,
                                                color: '#fff',
                                                fontSize: '12px'
                                            }}
                                        >
                                            {tag.tag_label}
                                        </span>
                                    ))}
                                </div>
                            </td>
                        </tr>
                    ))}
                </tbody>
            </table>
        </div>
    );
}
```

## Step 4: Verify GraphQL Schema

Merge the schema from `api/semantic_term_tags.graphql` into your main GraphQL schema:

```graphql
# In your main schema.graphql file

# Import all types from semantic_term_tags.graphql
include "semantic_term_tags.graphql"

# Extend SemanticTerm type with tags
extend type SemanticTerm {
    tags: [SemanticTag!]
    suggestedTags: [TagSuggestion!]
}

# Extend Query
extend type Query {
    semanticTags: [SemanticTag!]!
    tagsByCategory(category: String!): [SemanticTag!]!
    semanticTermTags(termId: ID!): [SemanticTag!]!
    tagCategories: [TagCategory!]!
    suggestSemanticTermTags(input: TagSuggestionInput!): TagSuggestionResponse!
}

# Extend Mutation
extend type Mutation {
    addTagToSemanticTerm(input: SemanticTermTagInput!): SemanticTerm!
    removeTagFromSemanticTerm(termId: ID!, tagKey: String!): SemanticTerm!
    updateSemanticTermTags(termId: ID!, tagKeys: [String!]!): SemanticTerm!
    createSemanticTag(input: TagInput!): SemanticTag!
    updateSemanticTag(tagKey: String!, input: TagInput!): SemanticTag!
    deleteSemanticTag(tagKey: String!): Boolean!
    acceptTagSuggestion(termId: ID!, tagKey: String!, isAccepted: Boolean!): SemanticTerm!
    applyTagSuggestions(termId: ID!, suggestedTags: [String!]!): SemanticTerm!
}
```

## Step 5: Testing the Integration

### Test Database Migration

```sql
-- Verify tables created
SELECT COUNT(*) as tag_count FROM semantic_term_tags;
-- Should return: 45

-- Verify tag categories
SELECT DISTINCT tag_category FROM semantic_term_tags ORDER BY tag_category;
-- Should show: business_area, data_type, domain, governance, sensitivity, usage_pattern

-- Check catalog_node schema
SELECT column_name, data_type FROM information_schema.columns 
WHERE table_name = 'catalog_node' AND column_name = 'tags';
-- Should show: tags, jsonb
```

### Test GraphQL Queries

```graphql
# Get all tags
query {
    semanticTags {
        id
        tagKey
        tagLabel
        tagCategory
        colorCode
    }
}

# Get tags by category
query {
    tagsByCategory(category: "business_area") {
        tagLabel
        tagCategory
    }
}

# Get tag suggestions for a new term
query {
    suggestSemanticTermTags(input: {
        nodeName: "total_revenue"
        displayName: "Total Revenue"
        description: "Total revenue by customer"
        dataType: "NUMERIC"
        domain: "sales"
        expression: "SUM(orders.amount)"
    }) {
        suggestions {
            tagKey
            tagLabel
            confidenceScore
            suggestionReason
        }
    }
}
```

### Test GraphQL Mutations

```graphql
# Apply tag suggestions
mutation {
    applyTagSuggestions(
        termId: "term-123"
        suggestedTags: ["numeric", "measure", "sales"]
    ) {
        id
        tags {
            tagKey
            tagLabel
        }
    }
}

# Create custom tag
mutation {
    createSemanticTag(input: {
        tagKey: "custom_business"
        tagLabel: "Custom Business"
        tagCategory: "business_area"
        colorCode: "#FF5733"
    }) {
        id
        tagKey
        tagLabel
    }
}
```

## Step 6: Example: Complete Semantic Term with Tags

### Creating a New Semantic Term with Wizard

```tsx
import React, { useState } from 'react';
import { SemanticTermTagsEditor, TagSuggestionWizard } from './components/SemanticTermTags/SemanticTermTags';

export function CreateSemanticTermWithTags() {
    const [formData, setFormData] = useState({
        nodeName: '',
        displayName: '',
        description: '',
        dataType: '',
        domain: '',
        expression: '',
    });
    
    const [tags, setTags] = useState<string[]>([]);
    const [showWizard, setShowWizard] = useState(false);
    
    const handleCreateTerm = async () => {
        // 1. Create semantic term via GraphQL
        const termResponse = await fetch('/api/graphql', {
            method: 'POST',
            body: JSON.stringify({
                query: `
                    mutation {
                        createSemanticTerm(input: {
                            nodeName: "${formData.nodeName}"
                            displayName: "${formData.displayName}"
                            description: "${formData.description}"
                            dataType: ${formData.dataType}
                            domain: "${formData.domain}"
                            expression: "${formData.expression}"
                        }) {
                            id
                        }
                    }
                `
            }),
        });
        
        const termResult = await termResponse.json();
        const termId = termResult.data.createSemanticTerm.id;
        
        // 2. Apply selected tags
        await fetch('/api/graphql', {
            method: 'POST',
            body: JSON.stringify({
                query: `
                    mutation {
                        applyTagSuggestions(
                            termId: "${termId}"
                            suggestedTags: ${JSON.stringify(tags)}
                        ) {
                            id
                            tags { tagKey tagLabel }
                        }
                    }
                `
            }),
        });
    };
    
    return (
        <div>
            <h2>Create Semantic Term</h2>
            
            {/* Form fields */}
            <input
                placeholder="Node Name"
                value={formData.nodeName}
                onChange={(e) => setFormData({...formData, nodeName: e.target.value})}
            />
            <input
                placeholder="Display Name"
                value={formData.displayName}
                onChange={(e) => setFormData({...formData, displayName: e.target.value})}
            />
            <textarea
                placeholder="Description"
                value={formData.description}
                onChange={(e) => setFormData({...formData, description: e.target.value})}
            />
            <select
                value={formData.dataType}
                onChange={(e) => setFormData({...formData, dataType: e.target.value})}
            >
                <option>NUMERIC</option>
                <option>STRING</option>
                <option>DATE</option>
                <option>BOOLEAN</option>
            </select>
            
            {/* Tag Editor */}
            <SemanticTermTagsEditor
                currentTags={tags}
                onTagsChange={setTags}
            />
            
            {/* Wizard Button */}
            <button onClick={() => setShowWizard(true)}>
                Get Tag Suggestions
            </button>
            
            {/* Tag Suggestion Wizard */}
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
            
            {/* Save Button */}
            <button onClick={handleCreateTerm}>Create Term</button>
        </div>
    );
}
```

## Predefined Tags Reference

The system includes 45 predefined tags across 6 categories:

### Business Area (10 tags)
- sales, finance, marketing, hr, operations, customer, product, supply_chain, legal, compliance

### Data Type (10 tags)
- numeric, text, date, boolean, currency, percentage, categorical, ordinal, interval, ratio

### Domain (10 tags)
- financial, healthcare, retail, manufacturing, utilities, education, government, technology, real_estate, agriculture

### Usage Pattern (8 tags)
- measure, dimension, derived_metric, kpi, fact, attribute, aggregate, calculated

### Sensitivity (4 tags)
- confidential, pii, sensitive, public

### Governance (3 tags)
- certified, regulated, deprecated

Each tag has:
- **Color Code**: For UI visualization (e.g., `#FF5733`)
- **Icon**: Font icon name for display
- **Auto Suggest**: Boolean for wizard inclusion
- **Sort Order**: Display ordering
- **Confidence Range**: 0.7-0.95 depending on inference source

## Troubleshooting

### Tags Not Appearing
1. Verify database migration executed: `SELECT COUNT(*) FROM semantic_term_tags;`
2. Check GraphQL resolvers are registered in schema
3. Verify React components imported correctly

### Suggestions Not Accurate
1. Check `tag_suggestion_service.go` inference methods
2. Verify term fields passed to suggestion query are populated
3. Check confidence thresholds in UI component

### Performance Issues
1. Add indexes on frequently queried columns (migration already includes these)
2. Implement pagination for large tag lists
3. Cache tag definitions in frontend

## Next Steps

1. ✅ Database migration created and ready
2. ✅ GraphQL resolvers implemented
3. ✅ React components created with styling
4. → Execute migration against your database
5. → Wire resolvers into your GraphQL server
6. → Integrate components into semantic term forms
7. → Test end-to-end workflow
8. → Optimize and deploy

