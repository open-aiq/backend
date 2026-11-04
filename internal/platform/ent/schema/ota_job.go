package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// OtaJob tracks one owner-approved firmware update of one device.
type OtaJob struct {
	ent.Schema
}

// Fields of the OtaJob.
func (OtaJob) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
		field.UUID("device_id", uuid.UUID{}).
			Immutable().
			Comment("Internal id of the device being updated"),
		field.UUID("release_id", uuid.UUID{}).
			Comment("Firmware release being installed"),
		field.Enum("status").
			Values("approved", "downloading", "installing", "succeeded", "failed", "cancelled").
			Default("approved").
			Comment("approved -> downloading -> installing -> succeeded | failed; cancelled before download"),
		field.Int("progress").
			Default(0).
			Min(0).
			Max(100).
			Comment("Download progress percentage reported by the device"),
		field.String("error").
			Optional().
			Nillable().
			MaxLen(500).
			Comment("Failure reason reported by the device"),
		field.String("approved_by").
			NotEmpty().
			Comment("Clerk user id that approved the update"),
		field.Time("approved_at").
			Default(time.Now),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Edges of the OtaJob.
func (OtaJob) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("device", Device.Type).
			Ref("ota_jobs").
			Field("device_id").
			Unique().
			Required().
			Immutable(),
		edge.From("release", FirmwareRelease.Type).
			Ref("ota_jobs").
			Field("release_id").
			Unique().
			Required(),
	}
}

// Indexes of the OtaJob.
func (OtaJob) Indexes() []ent.Index {
	return []ent.Index{
		// The latest job per device drives the device's OTA status.
		index.Fields("device_id", "created_at"),
	}
}
