package worker

import (
	"cert-proxy/cert-proxy-client/cert"
	"cert-proxy/internal/list"
	. "cert-proxy/internal/shared"
	"log"
	"sync"
)

// Task is a bundle of requests and associated destination
// files. Typically this is the bundle of operations necessary for
// one DN

type queue chan cert.Req
type Pool struct {
	wg *sync.WaitGroup
	queue
}

type fakeMutex struct{}

func (*fakeMutex) Lock()   {}
func (*fakeMutex) Unlock() {}

// NewPool creates worker pool of size workers and returns
// the Pool type. This pool type can be used to enqueue
// Tasks.
func NewPool(workers int) *Pool {

	var pool = Pool{
		wg:    new(sync.WaitGroup),
		queue: make(queue, workers),
	}
	var mtx = &sync.Mutex{}
	//var mtx = fakeMutex{}

	Verbose("Launching %d workers", workers)
	for i := 0; i < workers; i++ {
		pool.wg.Add(1)
		go func(wid int) {
			defer Verbose("Worker[%d] done", wid)
			defer pool.wg.Done()
			Verbose("Worker[%d] starting", wid)
			for req := range pool.queue {
				Verbose("Req %v\n", req)
				if err := req.Execute(mtx); err != nil {
					log.Print(err)
				}
			}

		}(i)
	}

	return &pool

}

// EnqueueTasks creates a task per CN, with all the items that are
// necessary for this CN. Bundling it per Domain has the advantage of a
// transaction like processing of the results
func (pool *Pool) EnqueueTasks(CNs list.UniqStrings, proxy, certbase, hook string, format cert.Format, pass string) {
	go func() {
		for cn := range CNs {
			if req, err := cert.NewReq(cn, proxy, certbase, hook, format, pass); err != nil {
				panic(err) // this is not supposed to fail
			} else {
				pool.queue <- req
			}
		}
		close(pool.queue)
	}()
}

// Wait until all jobs are done
func (pool Pool) Wait() {
	pool.wg.Wait()
}
