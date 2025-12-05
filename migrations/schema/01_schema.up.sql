CREATE TYPE campaign_status AS ENUM ('active', 'paused');
CREATE TYPE click_status AS ENUM ('allowed', 'fraud', 'error');

CREATE TABLE campaigns (
    campaign_id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    start_date TIMESTAMP NOT NULL,
    end_date TIMESTAMP NOT NULL,
    status campaign_status NOT NULL,
    target_url TEXT NOT NULL,
    link_id UUID UNIQUE NOT NULL
);

CREATE TABLE clicks (
    click_id UUID PRIMARY KEY,
    timestamp TIMESTAMP NOT NULL DEFAULT NOW(),
    link_id UUID NOT NULL,
    campaign_id UUID NOT NULL REFERENCES campaigns(campaign_id),
    user_id TEXT NOT NULL,
    ip_address TEXT,
    user_agent TEXT,
    referrer TEXT,
    device TEXT,
    device_model TEXT,
    browser TEXT,
    gaid TEXT,
    idfa TEXT,
    geo_country TEXT,
    geo_state TEXT,
    status click_status NOT NULL,
    fraud_check_failed TEXT[]
);

CREATE TABLE blocked_ids (
    id TEXT PRIMARY KEY,
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);