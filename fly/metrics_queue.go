package fly

import (
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var queueDepth atomic.Int64

var _ = promauto.NewGaugeFunc(
	prometheus.GaugeOpts{
		Name: "queue_depth",
		Help: "Generated value representing a queue depth.",
	},
	func() float64 { return float64(queueDepth.Load()) },
)

// This function walks back and forth between a range of values.
func Walk() {
	const min, max = 0, 100
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		slog.Info("queue depth increasing")
		for i := int64(min); i <= max; i++ {
			<-ticker.C
			queueDepth.Store(i)
		}

		slog.Info("queue depth decreasing")
		for i := int64(max); i >= min; i-- {
			<-ticker.C
			queueDepth.Store(i)
		}
	}
}
