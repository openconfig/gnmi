package queue

import (
	"sync"
	"time"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

// FixedQueue provides a strict delivery of provides updates.  If checkDelay,
// the Next() will sleep for the duration between the timestamps provided in the
// updates.
type FixedQueue struct {
	mu         sync.Mutex
	resp       []*gpb.SubscribeResponse
	delay      time.Duration
	checkDelay bool
}

// NewFixed creates a new FixedQueue with resp list of updates enqueued for
// iterating through.
func NewFixed(resp []*gpb.SubscribeResponse, delay bool) *FixedQueue {
	return &FixedQueue{
		resp:       resp,
		checkDelay: delay,
	}
}

// Add will append resp to the current tail of the queue.
func (q *FixedQueue) Add(resp *gpb.SubscribeResponse) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.resp = append(q.resp, resp)
}

// Next returns the next update in the queue or an error. If the queue is
// exhausted, a nil is returned for the update. The return will always be a
// *gpb.SubscribeResponse for proper type assertion.
func (q *FixedQueue) Next() (interface{}, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.resp) == 0 {
		return nil, nil
	}
	if q.delay != 0 {
		time.Sleep(q.delay)
	}
	resp := q.resp[0]
	q.resp = q.resp[1:]
	var n *gpb.SubscribeResponse_Update
	if len(q.resp) > 0 && q.checkDelay {
		var nOk bool
		n, nOk = resp.Response.(*gpb.SubscribeResponse_Update)
		next, nextOk := q.resp[0].Response.(*gpb.SubscribeResponse_Update)
		if !nOk || !nextOk {
			q.delay = 0
		} else {
			q.delay = time.Duration(next.Update.Timestamp-n.Update.Timestamp) * time.Nanosecond
			if q.delay < 0 {
				q.delay = 0
			}
		}
	}
	return resp, nil
}
