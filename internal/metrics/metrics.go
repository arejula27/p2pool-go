package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	SharechainHeight = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "p2pool",
		Name:      "sharechain_height",
		Help:      "Number of shares in the sharechain.",
	})

	MinersConnected = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "p2pool",
		Name:      "miners_connected",
		Help:      "Number of active stratum miner sessions.",
	})

	PeersConnected = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "p2pool",
		Name:      "peers_connected",
		Help:      "Number of connected P2P peers.",
	})

	ShareDifficulty = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "p2pool",
		Name:      "share_difficulty",
		Help:      "Current sharechain difficulty.",
	})

	PoolHashrate = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "p2pool",
		Name:      "pool_hashrate",
		Help:      "Estimated pool hashrate in H/s.",
	})

	LocalHashrate = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "p2pool",
		Name:      "local_hashrate",
		Help:      "Estimated local miner hashrate in H/s.",
	})

	BlocksFound = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "p2pool",
		Name:      "blocks_found_total",
		Help:      "Total Bitcoin blocks found by the pool.",
	})

	SharesAccepted = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "p2pool",
		Name:      "stratum_shares_accepted_total",
		Help:      "Total valid stratum shares accepted.",
	})

	SharesRejected = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "p2pool",
		Name:      "stratum_shares_rejected_total",
		Help:      "Total stratum shares rejected.",
	})

	BlockSubmissions = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "p2pool",
		Name:      "block_submissions_total",
		Help:      "Block submission attempts by result.",
	}, []string{"result"})

	UptimeSeconds = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "p2pool",
		Name:      "uptime_seconds",
		Help:      "Node uptime in seconds.",
	})
)

func init() {
	prometheus.MustRegister(
		SharechainHeight,
		MinersConnected,
		PeersConnected,
		ShareDifficulty,
		PoolHashrate,
		LocalHashrate,
		BlocksFound,
		SharesAccepted,
		SharesRejected,
		BlockSubmissions,
		UptimeSeconds,
	)
}

// Handler returns an HTTP handler for the /metrics endpoint.
func Handler() http.Handler {
	return promhttp.Handler()
}
