package health

import (
	"github.com/nylar/scout/health/healthpb"
)

type Status = healthpb.HealthCheckResponse_ServingStatus

// Health statuses
const (
	Unknown    = healthpb.HealthCheckResponse_UNKNOWN
	Serving    = healthpb.HealthCheckResponse_SERVING
	NotServing = healthpb.HealthCheckResponse_NOT_SERVING
)
