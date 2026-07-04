-- Modify "devices" table
ALTER TABLE "public"."devices" ADD COLUMN "is_outdoor" boolean NOT NULL DEFAULT false, ADD COLUMN "is_public" boolean NOT NULL DEFAULT false;
