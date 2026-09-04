-- owner_id intentionally remains nullable until every legacy device is assigned.
ALTER TABLE "devices" ADD COLUMN "owner_id" character varying NULL;
CREATE INDEX "device_owner_id_created_at" ON "devices" ("owner_id", "created_at");

-- One-time backfill (replace both values deliberately; repeat per owner):
-- UPDATE "devices" SET "owner_id" = 'user_...' WHERE "device_id" = 'dev_...';
-- Verify before creating a separate follow-up NOT NULL migration:
-- SELECT "device_id", "name" FROM "devices" WHERE "owner_id" IS NULL;
