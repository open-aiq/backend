-- Modify "devices" table
ALTER TABLE "public"."devices" ADD COLUMN "is_location_public" boolean NOT NULL DEFAULT false;
