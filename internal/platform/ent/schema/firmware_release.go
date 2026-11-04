package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// FirmwareRelease is a published device firmware image that devices can be
// updated to over the air. The binary itself is hosted elsewhere (e.g. a GitHub
// release asset); only its metadata is stored.
type FirmwareRelease struct {
	ent.Schema
}

// Fields of the FirmwareRelease.
func (FirmwareRelease) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
		field.String("version").
			NotEmpty().
			MaxLen(32).
			Unique().
			Immutable().
			Comment(`Semantic version without a leading "v", e.g. "0.2.0"`),
		field.String("url").
			NotEmpty().
			MaxLen(2048).
			Immutable().
			Comment("HTTPS URL of the OTA app image (.bin)"),
		field.String("sha256").
			NotEmpty().
			MaxLen(64).
			Immutable().
			Comment("Lowercase hex SHA-256 of the image"),
		field.Int64("size").
			Positive().
			Immutable().
			Comment("Image size in bytes"),
		field.Text("notes").
			Default("").
			Comment("Release notes shown to the owner before approving"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
	}
}

// Edges of the FirmwareRelease.
func (FirmwareRelease) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("ota_jobs", OtaJob.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}
