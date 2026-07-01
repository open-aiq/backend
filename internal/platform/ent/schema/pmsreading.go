package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// PMSReading holds the schema definition for a particulate matter sensor reading.
type PMSReading struct {
	ent.Schema
}

// Fields of the PMSReading.
func (PMSReading) Fields() []ent.Field {
	return []ent.Field{
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
		field.Int("aqi").
			NonNegative().
			Comment("Calculated Air Quality Index (US EPA scale; can exceed 500 in extreme pollution)"),
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			Comment("When the row was inserted"),
	}
}

// Indexes of the PMSReading.
func (PMSReading) Indexes() []ent.Index {
	return []ent.Index{
		// Time-series queries (historical / custom range) filter and sort by created_at.
		index.Fields("created_at"),
	}
}
