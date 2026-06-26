-- Process Templates Library Schema
-- This schema stores reusable workflow templates that users can browse, preview, and clone
-- Run: psql "postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable" -f backend/migrations/misc/process_templates_library_schema.sql

-- ============================================================================
-- MAIN TABLES
-- ============================================================================

-- Process templates catalog
CREATE TABLE IF NOT EXISTS process_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_key VARCHAR(100) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(50) NOT NULL, -- approval, data_collection, review, onboarding, compliance, other
    tags TEXT[], -- Searchable tags
    icon_name VARCHAR(50), -- Lucide icon name
    difficulty_level VARCHAR(20) DEFAULT 'beginner', -- beginner, intermediate, advanced
    estimated_setup_time_minutes INTEGER, -- Time to customize and deploy
    is_official BOOLEAN DEFAULT false, -- Official templates vs community
    is_featured BOOLEAN DEFAULT false, -- Featured on homepage
    
    -- Template content
    template_definition JSONB NOT NULL, -- Full BP process structure (steps, config, etc)
    customization_guide TEXT, -- Markdown guide for customizing
    example_use_cases TEXT[], -- Array of example scenarios
    
    -- Metadata
    author_name VARCHAR(255),
    author_organization VARCHAR(255),
    version VARCHAR(50) DEFAULT '1.0.0',
    compatible_with_version VARCHAR(50), -- System version compatibility
    
    -- Usage tracking
    usage_count INTEGER DEFAULT 0,
    clone_count INTEGER DEFAULT 0,
    favorite_count INTEGER DEFAULT 0,
    
    -- Ratings
    rating_average DECIMAL(3,2) DEFAULT 0.0, -- 0.00 to 5.00
    rating_count INTEGER DEFAULT 0,
    
    -- SEO and discovery
    search_keywords TEXT, -- Additional keywords for search
    documentation_url TEXT,
    demo_video_url TEXT,
    screenshot_url TEXT,
    
    -- Audit
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    published_at TIMESTAMP,
    
    -- Constraints
    CONSTRAINT valid_rating CHECK (rating_average >= 0 AND rating_average <= 5),
    CONSTRAINT valid_difficulty CHECK (difficulty_level IN ('beginner', 'intermediate', 'advanced')),
    CONSTRAINT valid_category CHECK (category IN ('approval', 'data_collection', 'review', 'onboarding', 'compliance', 'automation', 'notification', 'other'))
);

-- Template clones (tracks when users clone templates)
CREATE TABLE IF NOT EXISTS template_clones (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_id UUID NOT NULL REFERENCES process_templates(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    datasource_id UUID NOT NULL,
    process_id UUID, -- The created process ID (if available)
    
    -- User info
    cloned_by VARCHAR(255),
    
    -- Customization tracking
    was_customized BOOLEAN DEFAULT false,
    customization_notes TEXT,
    
    -- Timestamps
    cloned_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_used_at TIMESTAMP,
    
    -- For analytics
    time_to_first_use_minutes INTEGER, -- Time from clone to first execution
    usage_count INTEGER DEFAULT 0
);

-- Template ratings and reviews
CREATE TABLE IF NOT EXISTS template_ratings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_id UUID NOT NULL REFERENCES process_templates(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    datasource_id UUID NOT NULL,
    
    -- Rating
    rating INTEGER NOT NULL, -- 1-5 stars
    review_text TEXT,
    review_title VARCHAR(255),
    
    -- Reviewer info
    reviewer_name VARCHAR(255),
    reviewer_role VARCHAR(100),
    
    -- Helpful votes
    helpful_count INTEGER DEFAULT 0,
    not_helpful_count INTEGER DEFAULT 0,
    
    -- Status
    is_verified_user BOOLEAN DEFAULT false, -- Verified by usage tracking
    is_moderated BOOLEAN DEFAULT false,
    moderation_status VARCHAR(50) DEFAULT 'pending', -- pending, approved, rejected
    
    -- Audit
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    CONSTRAINT valid_rating_value CHECK (rating >= 1 AND rating <= 5),
    CONSTRAINT unique_rating_per_tenant UNIQUE(template_id, tenant_id, datasource_id)
);

-- Template categories metadata (for enhanced filtering)
CREATE TABLE IF NOT EXISTS template_categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    category_key VARCHAR(50) UNIQUE NOT NULL,
    display_name VARCHAR(100) NOT NULL,
    description TEXT,
    icon_name VARCHAR(50), -- Lucide icon name
    sort_order INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT true,
    template_count INTEGER DEFAULT 0, -- Cached count
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================================
-- INDEXES FOR PERFORMANCE
-- ============================================================================

-- Templates browsing and search
CREATE INDEX IF NOT EXISTS idx_templates_category ON process_templates(category, published_at DESC);
CREATE INDEX IF NOT EXISTS idx_templates_featured ON process_templates(is_featured, rating_average DESC) WHERE is_featured = true;
CREATE INDEX IF NOT EXISTS idx_templates_official ON process_templates(is_official, category) WHERE is_official = true;
CREATE INDEX IF NOT EXISTS idx_templates_rating ON process_templates(rating_average DESC, rating_count DESC);
CREATE INDEX IF NOT EXISTS idx_templates_usage ON process_templates(usage_count DESC, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_templates_tags ON process_templates USING GIN(tags);
CREATE INDEX IF NOT EXISTS idx_templates_search ON process_templates USING GIN(to_tsvector('english', name || ' ' || description || ' ' || COALESCE(search_keywords, '')));

-- Clones tracking
CREATE INDEX IF NOT EXISTS idx_clones_template ON template_clones(template_id, cloned_at DESC);
CREATE INDEX IF NOT EXISTS idx_clones_tenant ON template_clones(tenant_id, datasource_id, cloned_at DESC);
CREATE INDEX IF NOT EXISTS idx_clones_process ON template_clones(process_id) WHERE process_id IS NOT NULL;

-- Ratings
CREATE INDEX IF NOT EXISTS idx_ratings_template ON template_ratings(template_id, rating DESC, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ratings_moderation ON template_ratings(moderation_status, created_at) WHERE moderation_status = 'pending';

-- ============================================================================
-- TRIGGERS FOR AUTO-UPDATING
-- ============================================================================

-- Function to update timestamps
CREATE OR REPLACE FUNCTION update_template_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Triggers for timestamp updates
CREATE TRIGGER process_templates_updated
    BEFORE UPDATE ON process_templates
    FOR EACH ROW
    EXECUTE FUNCTION update_template_timestamp();

CREATE TRIGGER template_ratings_updated
    BEFORE UPDATE ON template_ratings
    FOR EACH ROW
    EXECUTE FUNCTION update_template_timestamp();

CREATE TRIGGER template_categories_updated
    BEFORE UPDATE ON template_categories
    FOR EACH ROW
    EXECUTE FUNCTION update_template_timestamp();

-- Function to update template ratings when a rating is added/updated/deleted
CREATE OR REPLACE FUNCTION update_template_rating_stats()
RETURNS TRIGGER AS $$
DECLARE
    avg_rating DECIMAL(3,2);
    total_count INTEGER;
BEGIN
    -- Calculate new average and count
    SELECT 
        COALESCE(AVG(rating), 0.0)::DECIMAL(3,2),
        COUNT(*)
    INTO avg_rating, total_count
    FROM template_ratings
    WHERE template_id = COALESCE(NEW.template_id, OLD.template_id)
      AND moderation_status = 'approved';
    
    -- Update template
    UPDATE process_templates
    SET 
        rating_average = avg_rating,
        rating_count = total_count,
        updated_at = NOW()
    WHERE id = COALESCE(NEW.template_id, OLD.template_id);
    
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- Triggers for rating updates
CREATE TRIGGER template_rating_inserted
    AFTER INSERT ON template_ratings
    FOR EACH ROW
    EXECUTE FUNCTION update_template_rating_stats();

CREATE TRIGGER template_rating_updated
    AFTER UPDATE ON template_ratings
    FOR EACH ROW
    WHEN (OLD.rating IS DISTINCT FROM NEW.rating OR OLD.moderation_status IS DISTINCT FROM NEW.moderation_status)
    EXECUTE FUNCTION update_template_rating_stats();

CREATE TRIGGER template_rating_deleted
    AFTER DELETE ON template_ratings
    FOR EACH ROW
    EXECUTE FUNCTION update_template_rating_stats();

-- ============================================================================
-- COMMENTS FOR DOCUMENTATION
-- ============================================================================

COMMENT ON TABLE process_templates IS 'Reusable workflow templates that users can browse, preview, and clone';
COMMENT ON TABLE template_clones IS 'Tracks when users clone templates into their tenant workspace';
COMMENT ON TABLE template_ratings IS 'User ratings and reviews for templates';
COMMENT ON TABLE template_categories IS 'Category metadata for organizing templates';

COMMENT ON COLUMN process_templates.template_definition IS 'Full BP process structure in JSONB format, ready to clone';
COMMENT ON COLUMN process_templates.customization_guide IS 'Markdown guide explaining how to customize this template';
COMMENT ON COLUMN process_templates.tags IS 'Array of searchable tags for discovery';
COMMENT ON COLUMN process_templates.difficulty_level IS 'Complexity level: beginner, intermediate, or advanced';
COMMENT ON COLUMN template_clones.was_customized IS 'Whether the user modified the template after cloning';
COMMENT ON COLUMN template_ratings.is_verified_user IS 'User has actually used the template (verified by usage tracking)';
