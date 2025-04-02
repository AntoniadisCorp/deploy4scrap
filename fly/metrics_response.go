package fly

import (
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var responseTimeMillis atomic.Int64 // Store response time in milliseconds

var _ = promauto.NewGaugeFunc(
	prometheus.GaugeOpts{
		Name: "response_time",
		Help: "Simulated HTTP response time in seconds.",
	},
	func() float64 {
		// Convert milliseconds to seconds for Prometheus
		return float64(responseTimeMillis.Load()) / 1000.0
	},
)

// This function walks back and forth between a range of response times.
func WalkResponse() {
	const min, max = 100, 2000 // Range from 0.1s (100ms) to 2.0s (2000ms)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		slog.Info("response time increasing")
		for t := int64(min); t <= max; t += 100 { // Increment by 0.1s (100ms)
			<-ticker.C
			responseTimeMillis.Store(t)
		}

		slog.Info("response time decreasing")
		for t := int64(max); t >= min; t -= 100 { // Decrement by 0.1s (100ms)
			<-ticker.C
			responseTimeMillis.Store(t)
		}
	}
}
