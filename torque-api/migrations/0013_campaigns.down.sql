DELETE FROM feature_permissions
 WHERE feature_key IN ('campaigns.view','campaigns.manage','campaigns.launch');

DROP TRIGGER IF EXISTS trg_campaign_recipients_updated_at ON campaign_recipients;
DROP INDEX IF EXISTS idx_campaign_recipients_org_lead;
DROP INDEX IF EXISTS idx_campaign_recipients_campaign_status;
DROP TABLE IF EXISTS campaign_recipients;
DROP TYPE IF EXISTS campaign_recipient_status;

DROP TRIGGER IF EXISTS trg_campaigns_updated_at ON campaigns;
DROP INDEX IF EXISTS idx_campaigns_org_scheduled;
DROP INDEX IF EXISTS idx_campaigns_org_status;
DROP TABLE IF EXISTS campaigns;
DROP TYPE IF EXISTS campaign_status;
