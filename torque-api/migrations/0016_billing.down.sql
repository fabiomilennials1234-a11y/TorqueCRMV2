-- Reverse S24.
DROP TABLE IF EXISTS billing_events;
DROP TABLE IF EXISTS subscriptions;
DROP TYPE IF EXISTS billing_provider;
DROP TYPE IF EXISTS subscription_status;

DELETE FROM feature_permissions
 WHERE feature_key IN ('billing.view','billing.checkout','billing.cancel');
