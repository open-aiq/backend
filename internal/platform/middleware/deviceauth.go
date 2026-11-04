package middleware

import (
	"context"
	"crypto/subtle"
	"errors"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"go-aiq-backend/internal/platform/problem"
)

// Header names a physical device authenticates with.
const (
	HeaderDeviceID        = "X-Device-ID"
	HeaderDeviceKey       = "X-Device-Key"
	HeaderFirmwareVersion = "X-Firmware-Version"
)

const deviceIDKey = "device_internal_id"

// ErrInvalidDeviceCredentials is returned by a DeviceAuthenticator when a
// device_id/device_key pair does not authenticate.
var ErrInvalidDeviceCredentials = errors.New("invalid device credentials")

// firmwareVersionRE accepts semantic versions such as "0.2.0" or "1.4.0-rc.1".
var firmwareVersionRE = regexp.MustCompile(`^v?\d{1,4}\.\d{1,4}\.\d{1,4}(-[0-9A-Za-z.-]{1,16})?$`)

// DeviceAuthenticator verifies device credentials and records the firmware
// version the device reported. Implemented by device.Service.
type DeviceAuthenticator interface {
	AuthenticateDevice(ctx context.Context, deviceID, deviceKey, firmwareVersion string) (uuid.UUID, error)
}

// DeviceAuth authenticates requests from physical devices using the
// X-Device-ID / X-Device-Key headers and stores the device's internal id in
// the context (see DeviceID). A well-formed X-Firmware-Version header is
// recorded; a malformed one is ignored so it can never block uploads.
func DeviceAuth(auth DeviceAuthenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		deviceID := c.GetHeader(HeaderDeviceID)
		deviceKey := c.GetHeader(HeaderDeviceKey)
		if deviceID == "" || deviceKey == "" {
			problem.Abort(c, problem.Unauthorized, HeaderDeviceID+" and "+HeaderDeviceKey+" headers are required.")
			return
		}

		id, err := auth.AuthenticateDevice(c.Request.Context(), deviceID, deviceKey, NormalizeFirmwareVersion(c.GetHeader(HeaderFirmwareVersion)))
		if err != nil {
			if errors.Is(err, ErrInvalidDeviceCredentials) {
				problem.Abort(c, problem.Unauthorized, "Invalid device credentials.")
				return
			}
			_ = c.Error(err)
			problem.Abort(c, problem.Internal, "Failed to authenticate device.")
			return
		}

		c.Set(deviceIDKey, id)
		c.Next()
	}
}

// DeviceID returns the internal id of the device authenticated by DeviceAuth.
func DeviceID(c *gin.Context) uuid.UUID {
	id, _ := c.Get(deviceIDKey)
	value, _ := id.(uuid.UUID)
	return value
}

// NormalizeFirmwareVersion trims a leading "v" and returns "" for values that
// are not semantic versions.
func NormalizeFirmwareVersion(value string) string {
	value = strings.TrimSpace(value)
	if !firmwareVersionRE.MatchString(value) {
		return ""
	}
	return strings.TrimPrefix(value, "v")
}

// StaticBearer authorizes requests carrying "Authorization: Bearer <token>"
// for a single shared secret (used by the firmware release publisher).
func StaticBearer(token string) gin.HandlerFunc {
	expected := []byte("Bearer " + token)
	return func(c *gin.Context) {
		got := []byte(c.GetHeader("Authorization"))
		if token == "" || subtle.ConstantTimeCompare(got, expected) != 1 {
			problem.Abort(c, problem.Unauthorized, "A valid publish token is required.")
			return
		}
		c.Next()
	}
}
