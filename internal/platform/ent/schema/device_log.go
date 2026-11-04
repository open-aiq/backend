package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// DeviceLog is one important event (warning, error, boot, WiFi, OTA, ...) a
// device uploaded for remote troubleshooting.
type DeviceLog struct {
	ent.Schema
}

// Fields of the DeviceLog.
func (DeviceLog) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").
			Immutable(),
		field.UUID("device_id", uuid.UUID{}).
			Immutable(),
		field.Enum("level").
			Values("debug", "info", "warn", "error").
			Immutable(),
		field.String("tag").
			MaxLen(32).
			Default("").
			Immutable().
			Comment("Subsystem that produced the entry, e.g. wifi, ota, upload"),
		field.String("message").
			NotEmpty().
			MaxLen(256).
			Immutable(),
		field.Int64("uptime_ms").
			Min(0).
			Immutable().
			Comment("Milliseconds since the device booted when the entry was recorded"),
		field.Int("boot_count").
			Min(0).
			Immutable().
			Comment("Device boot counter; groups entries by power cycle"),
		field.Time("device_ts").
			Optional().
			Nillable().
			Immutable().
			Comment("Device wall-clock time (SNTP); null when the clock was not synced"),
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			Comment("When the backend received the entry"),
	}
}

// Edges of the DeviceLog.
func (DeviceLog) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("device", Device.Type).
			Ref("logs").
			Field("device_id").
			Unique().
			Required().
			Immutable(),
	}
}

// Indexes of the DeviceLog.
func (DeviceLog) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("device_id", "id"),
		// Retention purges by age.
		index.Fields("created_at"),
	}
}
