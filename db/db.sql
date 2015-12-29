BEGIN;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE TABLE IF NOT EXISTS endpoints (
  id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  organization_id   UUID NOT NULL,
  schema            TEXT,
  url               TEXT NOT NULL
);
COMMENT ON TABLE endpoints IS 'API endpoints registered with the service';
COMMENT ON COLUMN endpoints.id IS 'UUID of endpoint';
COMMENT ON COLUMN endpoints.organization_id IS 'Organization which endpoint belongs to';
COMMENT ON COLUMN endpoints.url IS 'URL of endpoint';
COMMIT;
