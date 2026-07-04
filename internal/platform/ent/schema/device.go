package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// Device holds the schema definition for a registered air quality device.
type Device struct {
	ent.Schema
}

// Fields of the Device.
func (Device) Fields() []ent.Field {
	return []ent.Field{
		// Internal primary key. Universally unique, never exposed in the API.
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable().
			Comment("Internal primary key; never exposed in API responses"),
		// Public, opaque device identifier that clients integrate against,
		// e.g. "dev_9f1c8a3b...". Independent from the internal PK so it can be
		// reissued later without touching foreign keys.
		field.String("device_id").
			NotEmpty().
			Immutable().
			Unique().
			Comment(`Public device identifier, e.g. "dev_<uuid>"`),
		field.String("name").
			NotEmpty().
			Comment("Human-readable device name"),
		field.Bool("is_outdoor").
			Default(false).
			Comment("Whether the device is installed outdoors"),
		field.Bool("is_public").
			Default(false).
			Comment("Whether the owner shares this device's data publicly"),
		field.String("device_key").
			NotEmpty().
			Sensitive().
			Comment(`Hex SHA-256 digest of the secret device key ("sk_<random>"); the raw key is shown once at creation/rotation and never stored`),
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			Comment("When the device was registered"),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now).
			Comment("When the device was last updated"),
	}
}

// Edges of the Device.
func (Device) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("readings", DeviceReading.Type),
	}
}

// Indexes of the Device.
func (Device) Indexes() []ent.Index {
	return []ent.Index{
		// Listings are ordered by registration time.
		index.Fields("created_at"),
	}
}
