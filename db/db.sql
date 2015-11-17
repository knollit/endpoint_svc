BEGIN;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE TABLE IF NOT EXISTS endpoints (
  id            UUID PRIMARY KEY DEFAULT uuid_generate_v1mc(),
  organization  TEXT NOT NULL,
  url           TEXT NOT NULL
);
COMMENT ON TABLE endpoints IS 'API endpoints registered with the service';
COMMENT ON COLUMN endpoints.id IS 'UUID of endpoint';
COMMENT ON COLUMN endpoints.organization IS 'Organization which endpoint belongs to';
COMMENT ON COLUMN endpoints.url IS 'URL of endpoint';
COMMIT;
