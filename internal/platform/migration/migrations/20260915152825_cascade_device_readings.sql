-- Modify "device_readings" table
ALTER TABLE "public"."device_readings" DROP CONSTRAINT "device_readings_devices_readings", ADD CONSTRAINT "device_readings_devices_readings" FOREIGN KEY ("device_id") REFERENCES "public"."devices" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
