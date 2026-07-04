package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// DeviceReading holds the schema definition for a full sensor reading uploaded
// by a device: particulate matter, calculated AQI, temperature, and an optional
// location fix. One row per device upload.
type DeviceReading struct {
	ent.Schema
}

// Fields of the DeviceReading.
func (DeviceReading) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("device_id", uuid.UUID{}).
			Immutable().
			Comment("Internal id of the device that uploaded this reading"),
		field.Float("pm1_0").
			Min(0).
			StructTag(`json:"pm1_0"`).
			Comment("PM1.0 concentration (µg/m³)"),
		field.Float("pm2_5").
			Min(0).
			StructTag(`json:"pm2_5"`).
			Comment("PM2.5 concentration (µg/m³)"),
		field.Float("pm10_0").
			Min(0).
			StructTag(`json:"pm10_0"`).
			Comment("PM10 concentration (µg/m³)"),
		field.String("pms_provider").
			NotEmpty().
			Comment(`Particulate matter sensor model, e.g. "pms5003"`),
		field.Int("aqi").
			NonNegative().
			Comment("Device-calculated Air Quality Index (US EPA scale; can exceed 500 in extreme pollution)"),
		field.Float("temperature").
			Comment("Temperature (°C)"),
		field.Float("humidity").
			Min(0).
			Max(100).
			Comment("Relative humidity (%)"),
		field.Float("heat_index").
			Comment("Heat index / feels-like temperature (°C)"),
		field.String("temperature_provider").
			NotEmpty().
			Comment(`Temperature sensor model, e.g. "dht22"`),
		field.Float("lat").
			Optional().
			Nillable().
			Min(-90).
			Max(90).
			Comment("Latitude when the reading was taken; absent if no location fix"),
		field.Float("lon").
			Optional().
			Nillable().
			Min(-180).
			Max(180).
			Comment("Longitude when the reading was taken; absent if no location fix"),
		field.String("location_provider").
			Optional().
			Comment(`Source of the location fix, e.g. "mobile"; absent if no location fix`),
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			Comment("When the row was inserted"),
	}
}

// Edges of the DeviceReading.
func (DeviceReading) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("device", Device.Type).
			Ref("readings").
			Field("device_id").
			Unique().
			Required().
			Immutable(),
	}
}

// Indexes of the DeviceReading.
func (DeviceReading) Indexes() []ent.Index {
	return []ent.Index{
		// Time-series queries (current / historical) filter and sort by created_at.
		index.Fields("created_at"),
		// Per-device time-series queries.
		index.Fields("device_id", "created_at"),
	}
}
