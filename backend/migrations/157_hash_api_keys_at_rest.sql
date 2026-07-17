-- Store API keys as one-way SHA-256 identifiers. The short prefix/suffix are
-- retained only for UI identification and are not sufficient for authentication.
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '10min';

UPDATE api_keys
SET key = 'sha256$' || encode(sha256(convert_to(key, 'UTF8')), 'hex') || '$' ||
          encode(substring(convert_to(key, 'UTF8') from 1 for 6), 'hex') || '$' ||
          encode(substring(convert_to(key, 'UTF8') from greatest(octet_length(convert_to(key, 'UTF8')) - 3, 1)), 'hex')
WHERE deleted_at IS NULL
  AND key NOT LIKE 'sha256$%';

UPDATE deleted_api_key_audits
SET key = 'sha256$' || encode(sha256(convert_to(key, 'UTF8')), 'hex') || '$' ||
          encode(substring(convert_to(key, 'UTF8') from 1 for 6), 'hex') || '$' ||
          encode(substring(convert_to(key, 'UTF8') from greatest(octet_length(convert_to(key, 'UTF8')) - 3, 1)), 'hex')
WHERE key NOT LIKE 'sha256$%';
