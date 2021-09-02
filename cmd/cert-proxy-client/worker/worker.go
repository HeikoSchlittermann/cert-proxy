package worker

import (
	"fmt"
	"go.schlittermann.de/heiko/cert-proxy.git/cmd/cert-proxy-client/cert"
	"go.schlittermann.de/heiko/cert-proxy.git/list"
	. "go.schlittermann.de/heiko/cert-proxy.git/shared"
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
	errors chan int
}

type fakeMutex struct{}

func (*fakeMutex) Lock()   {}
func (*fakeMutex) Unlock() {}

// NewPool creates worker pool of size workers and returns
// the Pool type. This pool type can be used to enqueue
// Tasks.
func NewPool(workers int) *Pool {

	var pool = Pool{
		wg:     new(sync.WaitGroup),
		queue:  make(queue, workers),
		errors: make(chan int),
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
					log.Println(err)
					pool.errors <- 1
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
func (pool Pool) Wait() error {

	var done = make(chan int)

	// Collect and output the error messages
	go func() {
		var errors int
		for n := range pool.errors {
			errors += n
		}
		done <- errors
	}()

	pool.wg.Wait()     // for for all workers to complete
	close(pool.errors) // this terminates the above goroutine, but we don't care to wait for it
	errors := <-done

	if errors != 0 {
		return fmt.Errorf("Got %d error%s", errors, plural(errors))
	} else {
		return nil
	}
}

func plural(i int) (s string) {
	if i != 1 {
		s = `s`
	}
	return
}
