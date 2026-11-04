-- Create "firmware_releases" table
CREATE TABLE "public"."firmware_releases" (
  "id" uuid NOT NULL,
  "version" character varying NOT NULL,
  "url" character varying NOT NULL,
  "sha256" character varying NOT NULL,
  "size" bigint NOT NULL,
  "notes" text NOT NULL DEFAULT '',
  "created_at" timestamptz NOT NULL,
  PRIMARY KEY ("id")
);
-- Create index "firmware_releases_version_key" to table: "firmware_releases"
CREATE UNIQUE INDEX "firmware_releases_version_key" ON "public"."firmware_releases" ("version");
-- Create "ota_jobs" table
CREATE TABLE "public"."ota_jobs" (
  "id" uuid NOT NULL,
  "status" character varying NOT NULL DEFAULT 'approved',
  "progress" bigint NOT NULL DEFAULT 0,
  "error" character varying NULL,
  "approved_by" character varying NOT NULL,
  "approved_at" timestamptz NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "device_id" uuid NOT NULL,
  "release_id" uuid NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "ota_jobs_devices_ota_jobs" FOREIGN KEY ("device_id") REFERENCES "public"."devices" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "ota_jobs_firmware_releases_ota_jobs" FOREIGN KEY ("release_id") REFERENCES "public"."firmware_releases" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create index "otajob_device_id_created_at" to table: "ota_jobs"
CREATE INDEX "otajob_device_id_created_at" ON "public"."ota_jobs" ("device_id", "created_at");
