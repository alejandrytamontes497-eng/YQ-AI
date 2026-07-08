-- When registration email verification is enabled, first-time third-party
-- signups must also confirm an email before account creation.

INSERT INTO settings (key, value, updated_at)
SELECT 'force_email_on_third_party_signup', 'true', NOW()
WHERE EXISTS (
  SELECT 1
  FROM settings
  WHERE key = 'email_verify_enabled'
    AND value = 'true'
)
ON CONFLICT (key) DO UPDATE
SET value = 'true',
    updated_at = NOW()
WHERE EXISTS (
  SELECT 1
  FROM settings
  WHERE key = 'email_verify_enabled'
    AND value = 'true'
);
