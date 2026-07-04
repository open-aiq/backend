#!/usr/bin/env bash
#
# seed_readings.sh — seed mock sensor readings for a device.
#
# Inserts realistic device_readings rows directly into the dev database via the
# openaiq-postgres container (the API server-assigns created_at, so historical
# data cannot be backdated through it). Three density tiers keep the row count
# sane while covering every chart timeline:
#
#   - last 24 hours:      every 10 minutes  (matches the real sampling rate)
#   - previous 30 days:   hourly
#   - previous 12 months: every 6 hours
#
# PM2.5 follows a daily sine wave plus noise; PM1.0/PM10 and AQI are derived
# from it; temperature peaks mid-afternoon. A final row carries a location fix.
#
# Intended to be run via `make seed device=dev_...`, which passes the
# environment below.
#
set -euo pipefail

DEVICE_ID="${DEVICE_ID:-}"
DATABASE_URL="${DATABASE_URL:-}"
CONTAINER="${CONTAINER:-openaiq-postgres}"

die() {
	echo "error: $*" >&2
	exit 1
}

# --- Preflight ---------------------------------------------------------------

[ -n "$DEVICE_ID" ] || die "DEVICE_ID is required (public device id, e.g. dev_...)"
[ -n "$DATABASE_URL" ] || die "DATABASE_URL is required (defined in .env)"

docker inspect "$CONTAINER" >/dev/null 2>&1 ||
	die "container '$CONTAINER' is not running. Start it with: make db-up"

# Derive the psql user/database from DATABASE_URL (postgres://user:pass@host:port/db?...).
creds="${DATABASE_URL#*://}"
db_user="${creds%%:*}"
path="${DATABASE_URL##*/}"
db_name="${path%%\?*}"

# --- Seed --------------------------------------------------------------------

# :'device_id' is injected via --set; \gset aborts (with ON_ERROR_STOP) if the
# device is unknown.
docker exec -i "$CONTAINER" psql \
	--username "$db_user" \
	--dbname "$db_name" \
	--set ON_ERROR_STOP=1 \
	--set device_id="$DEVICE_ID" \
	--quiet <<'SQL'
SELECT id AS dev_id FROM devices WHERE device_id = :'device_id' \gset

-- One reusable shape per tier: pm2_5 = base daily sine wave + noise;
-- everything else is derived from pm2_5 / time of day.
INSERT INTO device_readings
	(device_id, pm1_0, pm2_5, pm10_0, pms_provider, aqi,
	 temperature, humidity, heat_index, temperature_provider, created_at)
SELECT
	:'dev_id'::uuid,
	round((pm2_5 * 0.6)::numeric, 1),
	round(pm2_5::numeric, 1),
	round((pm2_5 * 1.5)::numeric, 1),
	'pms5003',
	greatest(0, round(pm2_5 * 2.1))::bigint,
	round(temp::numeric, 1),
	round((35 + random() * 50)::numeric, 1),
	round((temp + 3 + random() * 3)::numeric, 1),
	'dht22',
	ts
FROM (
	SELECT
		ts,
		greatest(2, 25 + 20 * sin(2 * pi() * extract(epoch FROM ts) / 86400) + random() * 15) AS pm2_5,
		28 + 5 * sin(2 * pi() * (extract(epoch FROM ts) - 50400) / 86400) + random() * 2      AS temp
	FROM (
		          SELECT generate_series(now() - interval '24 hours',  now(),                       interval '10 minutes') AS ts
		UNION ALL SELECT generate_series(now() - interval '30 days',   now() - interval '24 hours', interval '1 hour')     AS ts
		UNION ALL SELECT generate_series(now() - interval '12 months', now() - interval '30 days',  interval '6 hours')    AS ts
	) AS series
) AS sample;

-- Latest reading carries a location fix (Karachi) from a paired mobile.
INSERT INTO device_readings
	(device_id, pm1_0, pm2_5, pm10_0, pms_provider, aqi,
	 temperature, humidity, heat_index, temperature_provider,
	 lat, lon, location_provider, created_at)
VALUES
	(:'dev_id'::uuid, 18.2, 30.4, 45.8, 'pms5003', 64,
	 31.5, 58.0, 35.9, 'dht22',
	 24.8607, 67.0011, 'mobile', now());

SELECT count(*) AS total_readings FROM device_readings WHERE device_id = :'dev_id'::uuid;
SQL

echo "Seeded readings for $DEVICE_ID."
