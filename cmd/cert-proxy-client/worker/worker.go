// Copyright 2019-2024 Heiko Schlittermann <hs@schlittermann.de>
// SPDX-License-Identifier: Apache-2.0

// Package worker implements a fixed-size pool that runs cert.Req fetches
// concurrently across many domains.
package worker

import (
	"fmt"
	"log"
	"sync"

	"go.schlittermann.de/heiko/cert-proxy/cmd/cert-proxy-client/cert"
	"go.schlittermann.de/heiko/cert-proxy/internal/list"
	"go.schlittermann.de/heiko/cert-proxy/internal/shared"
)

// Task is a bundle of requests and associated destination
// files. Typically this is the bundle of operations necessary for
// one DN

type queue chan cert.Req

// Pool is a fixed-size set of workers that execute cert.Req fetches.
type Pool struct {
	wg *sync.WaitGroup
	queue
	errors chan int
}

// NewPool creates worker pool of size workers and returns
// the Pool. This pool can be used to enqueue
// Tasks.
func NewPool(workers int) *Pool {
	var (
		pool = Pool{wg: new(sync.WaitGroup), queue: make(queue, workers), errors: make(chan int)}
		mtx  = &sync.Mutex{}
	)

	shared.Verbose("Launching %d workers", workers)

	for i := 0; i < workers; i++ {
		pool.wg.Add(1)

		go func(wid int) {
			defer shared.Verbose("Worker[%d] done", wid)
			defer pool.wg.Done()

			shared.Verbose("Worker[%d] starting", wid)

			for req := range pool.queue {
				shared.Verbose("Req %v\n", req)

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
			req, err := cert.NewReq(cn, proxy, certbase, hook, format, pass)
			if err != nil {
				panic(err) // this is not supposed to fail
			}

			pool.queue <- req
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
		return fmt.Errorf("got %d error%s", errors, plural(errors))
	}

	return nil
}

func plural(i int) (s string) {
	if i != 1 {
		s = `s`
	}

	return
}
