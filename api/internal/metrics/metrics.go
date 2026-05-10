package metrics

import (
	"fmt"
	"sync/atomic"
)

type Counters struct {
	RequestsTotal uint64
	AcceptedTotal uint64
	ErrorsTotal   uint64
}

func (c *Counters) IncRequests() { atomic.AddUint64(&c.RequestsTotal, 1) }
func (c *Counters) IncAccepted() { atomic.AddUint64(&c.AcceptedTotal, 1) }
func (c *Counters) IncErrors()   { atomic.AddUint64(&c.ErrorsTotal, 1) }

func (c *Counters) Prometheus() string {
	return fmt.Sprintf(`# HELP kyc_requests_total Total HTTP KYC requests.
# TYPE kyc_requests_total counter
kyc_requests_total %d
# HELP kyc_accepted_total Total accepted async KYC jobs.
# TYPE kyc_accepted_total counter
kyc_accepted_total %d
# HELP kyc_errors_total Total API errors.
# TYPE kyc_errors_total counter
kyc_errors_total %d
`, atomic.LoadUint64(&c.RequestsTotal), atomic.LoadUint64(&c.AcceptedTotal), atomic.LoadUint64(&c.ErrorsTotal))
}
