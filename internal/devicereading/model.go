package devicereading

import "time"

// Numeric fields in the request DTOs are pointers so a legitimate zero value
// (e.g. pm2_5 = 0 in clean air) still passes `required` validation.

// PMSData is the particulate matter section of an upload.
type PMSData struct {
	PM10     *float64 `json:"pm1_0" validate:"required,gte=0" example:"5.2"`
	PM25     *float64 `json:"pm2_5" validate:"required,gte=0" example:"12.1"`
	PM100    *float64 `json:"pm10_0" validate:"required,gte=0" example:"18.3"`
	Provider string   `json:"provider" mod:"trim" validate:"required,max=50,provider" example:"pms5003"`
}

// TemperatureData is the temperature section of an upload.
type TemperatureData struct {
	Temperature *float64 `json:"temperature" validate:"required" example:"31.2"`
	Humidity    *float64 `json:"humidity" validate:"required,gte=0,lte=100" example:"64.5"`
	HeatIndex   *float64 `json:"heat_index" validate:"required" example:"36.8"`
	Provider    string   `json:"provider" mod:"trim" validate:"required,max=50,provider" example:"dht22"`
}

// Location is the optional location fix of an upload, provided by a paired mobile.
type Location struct {
	Lat      *float64 `json:"lat" validate:"required,gte=-90,lte=90" example:"24.8607"`
	Lon      *float64 `json:"lon" validate:"required,gte=-180,lte=180" example:"67.0011"`
	Provider string   `json:"provider" mod:"trim" validate:"required,max=50,provider" example:"mobile"`
}

// UploadRequest is the payload a device sends to POST /data.
type UploadRequest struct {
	PMSData         *PMSData         `json:"pms_data" validate:"required"`
	AQI             *int             `json:"aqi" validate:"required,gte=0" example:"57"`
	TemperatureData *TemperatureData `json:"temperature_data" validate:"required"`
	// Location is optional: the device may have no paired mobile at upload time.
	Location *Location `json:"location"`
}

// UploadedReading acknowledges a stored reading.
type UploadedReading struct {
	ID        int       `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

// Response types for swagger documentation.

type UploadResponse struct {
	Data UploadedReading `json:"data"`
}
