-- services/backlog-service/migrations/000001_initial_schema.up.sql

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create enum types
CREATE TYPE item_type AS ENUM ('EPIC', 'FEATURE', 'STORY');
CREATE TYPE item_status AS ENUM ('NEW', 'READY', 'IN_PROGRESS', 'DONE', 'BLOCKED');

-- Create backlog items table
CREATE TABLE backlog_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    type item_type NOT NULL,
    parent_id UUID REFERENCES backlog_items(id) ON DELETE SET NULL,
    title TEXT NOT NULL,
    description TEXT,
    story_points INTEGER NOT NULL DEFAULT 0,
    status item_status NOT NULL DEFAULT 'NEW',
    priority INTEGER NOT NULL DEFAULT 0,
    assignee TEXT,
    tags TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    external_ids JSONB NOT NULL DEFAULT '{}'::JSONB,
    
    -- Add constraints
    CONSTRAINT backlog_items_title_not_empty CHECK (length(trim(title)) > 0),
    CONSTRAINT backlog_items_story_points_not_negative CHECK (story_points >= 0)
);

-- Create index on parent_id for fast child lookup
CREATE INDEX idx_backlog_items_parent_id ON backlog_items(parent_id);

-- Create index on type for filtering
CREATE INDEX idx_backlog_items_type ON backlog_items(type);

-- Create index on status for filtering
CREATE INDEX idx_backlog_items_status ON backlog_items(status);

-- Create index on priority for sorting
CREATE INDEX idx_backlog_items_priority ON backlog_items(priority);

-- Create index on assignee for filtering
CREATE INDEX idx_backlog_items_assignee ON backlog_items(assignee);

-- Create index on tags for filtering
CREATE INDEX idx_backlog_items_tags ON backlog_items USING GIN(tags);

-- Create index on external_ids for lookup by external ID
CREATE INDEX idx_backlog_items_external_ids ON backlog_items USING GIN(external_ids);

-- Create function to automatically update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger to automatically update updated_at timestamp
CREATE TRIGGER update_backlog_items_updated_at
BEFORE UPDATE ON backlog_items
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- Create events table for event sourcing
CREATE TABLE events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_type TEXT NOT NULL,
    item_id UUID REFERENCES backlog_items(id) ON DELETE CASCADE,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create index on item_id for fast event lookup by item
CREATE INDEX idx_events_item_id ON events(item_id);

-- Create index on event_type for filtering
CREATE INDEX idx_events_event_type ON events(event_type);

-- Create index on created_at for time-based queries
CREATE INDEX idx_events_created_at ON events(created_at);

-- Create comments table
CREATE TABLE comments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    item_id UUID NOT NULL REFERENCES backlog_items(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Add constraints
    CONSTRAINT comments_content_not_empty CHECK (length(trim(content)) > 0)
);

-- Create index on item_id for fast comment lookup by item
CREATE INDEX idx_comments_item_id ON comments(item_id);

-- Create index on user_id for filtering
CREATE INDEX idx_comments_user_id ON comments(user_id);

-- Create trigger to automatically update updated_at timestamp
CREATE TRIGGER update_comments_updated_at
BEFORE UPDATE ON comments
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- Create metric_snapshots table for storing historical metrics
CREATE TABLE metric_snapshots (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    total_items INTEGER NOT NULL,
    epic_count INTEGER NOT NULL,
    feature_count INTEGER NOT NULL,
    story_count INTEGER NOT NULL,
    average_age FLOAT NOT NULL,
    wip_count INTEGER NOT NULL,
    lead_time_days FLOAT NOT NULL,
    throughput_last_30_days INTEGER NOT NULL,
    iceberg_ratio FLOAT NOT NULL,
    health_status TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create index on created_at for time-based queries
CREATE INDEX idx_metric_snapshots_created_at ON metric_snapshots(created_at);

-- Create history table for tracking item status changes
CREATE TABLE item_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    item_id UUID NOT NULL REFERENCES backlog_items(id) ON DELETE CASCADE,
    field_name TEXT NOT NULL,
    old_value TEXT,
    new_value TEXT NOT NULL,
    user_id TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create index on item_id for fast history lookup by item
CREATE INDEX idx_item_history_item_id ON item_history(item_id);

-- Create index on field_name for filtering
CREATE INDEX idx_item_history_field_name ON item_history(field_name);

-- Create workshops table for tracking workshops
CREATE TABLE workshops (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT NOT NULL,
    description TEXT,
    workshop_type TEXT NOT NULL,
    facilitator TEXT NOT NULL,
    scheduled_at TIMESTAMPTZ NOT NULL,
    duration_minutes INTEGER NOT NULL,
    status TEXT NOT NULL DEFAULT 'SCHEDULED',
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Add constraints
    CONSTRAINT workshops_name_not_empty CHECK (length(trim(name)) > 0),
    CONSTRAINT workshops_duration_positive CHECK (duration_minutes > 0)
);

-- Create workshop_items table for tracking items discussed in workshops
CREATE TABLE workshop_items (
    workshop_id UUID NOT NULL REFERENCES workshops(id) ON DELETE CASCADE,
    item_id UUID NOT NULL REFERENCES backlog_items(id) ON DELETE CASCADE,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    PRIMARY KEY (workshop_id, item_id)
);

-- Create index on workshop_id for fast item lookup by workshop
CREATE INDEX idx_workshop_items_workshop_id ON workshop_items(workshop_id);

-- Create index on item_id for fast workshop lookup by item
CREATE INDEX idx_workshop_items_item_id ON workshop_items(item_id);

-- Create trigger to automatically update updated_at timestamp
CREATE TRIGGER update_workshops_updated_at
BEFORE UPDATE ON workshops
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- Create view for backlog health metrics
CREATE OR REPLACE VIEW backlog_health AS
SELECT
    COUNT(*) FILTER (WHERE status != 'DONE') AS total_active_items,
    COUNT(*) FILTER (WHERE type = 'EPIC' AND status != 'DONE') AS active_epics,
    COUNT(*) FILTER (WHERE type = 'FEATURE' AND status != 'DONE') AS active_features,
    COUNT(*) FILTER (WHERE type = 'STORY' AND status != 'DONE') AS active_stories,
    COUNT(*) FILTER (WHERE status = 'IN_PROGRESS') AS wip_count,
    COUNT(*) FILTER (WHERE status = 'BLOCKED') AS blocked_count,
    COUNT(*) FILTER (WHERE status = 'NEW' AND created_at < NOW() - INTERVAL '30 days') AS ageing_items,
    AVG(EXTRACT(EPOCH FROM (NOW() - created_at)) / 86400) FILTER (WHERE status = 'NEW') AS avg_new_age_days,
    AVG(EXTRACT(EPOCH FROM (updated_at - created_at)) / 86400) FILTER (WHERE status = 'DONE' AND updated_at > NOW() - INTERVAL '30 days') AS avg_lead_time_days,
    COUNT(*) FILTER (WHERE status = 'DONE' AND updated_at > NOW() - INTERVAL '30 days') AS completed_last_30_days
FROM
    backlog_items;

-- Create function to verify parent-child relationship
CREATE OR REPLACE FUNCTION validate_parent_child_relationship()
RETURNS TRIGGER AS $$
DECLARE
    parent_type item_type;
BEGIN
    -- Skip validation if parent_id is NULL
    IF NEW.parent_id IS NULL THEN
        RETURN NEW;
    END IF;
    
    -- Get parent type
    SELECT type INTO parent_type
    FROM backlog_items
    WHERE id = NEW.parent_id;
    
    -- Validate relationship
    IF (parent_type = 'EPIC' AND NEW.type = 'FEATURE') OR
       (parent_type = 'FEATURE' AND NEW.type = 'STORY') THEN
        RETURN NEW;
    ELSE
        RAISE EXCEPTION 'Invalid parent-child relationship: % cannot be a parent of %', parent_type, NEW.type;
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Create trigger to validate parent-child relationship
CREATE TRIGGER validate_backlog_item_parent_child
BEFORE INSERT OR UPDATE OF parent_id, type ON backlog_items
FOR EACH ROW
WHEN (NEW.parent_id IS NOT NULL)
EXECUTE FUNCTION validate_parent_child_relationship();
