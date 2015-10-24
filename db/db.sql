BEGIN;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE TABLE endpoints (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v1mc(),
  watchpointURL text
);
COMMENT ON TABLE endpoints IS 'API endpoints registered with the service';
COMMENT ON COLUMN endpoints.id IS 'UUID of endpoint';
COMMENT ON COLUMN endpoints.watchpointURL IS 'URL of endpoint schema to monitor';
COMMIT;