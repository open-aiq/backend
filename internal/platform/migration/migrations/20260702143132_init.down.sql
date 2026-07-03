-- reverse: create index "pmsreading_created_at" to table: "pms_readings"
DROP INDEX "pmsreading_created_at";
-- reverse: create "pms_readings" table
DROP TABLE "pms_readings";
-- reverse: create index "devices_device_id_key" to table: "devices"
DROP INDEX "devices_device_id_key";
-- reverse: create index "device_created_at" to table: "devices"
DROP INDEX "device_created_at";
-- reverse: create "devices" table
DROP TABLE "devices";
