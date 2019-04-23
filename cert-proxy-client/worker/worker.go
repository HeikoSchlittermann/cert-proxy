package worker

import (
	"cert-proxy/cert-proxy-client/cert"
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
					log.Fatal(err)
				}
			}

		}(i)
	}

	return &pool

}

// EnqueueTasks creates a task per CN, with all the items that are
// necessary for this CN. Bundling it per Domain has the advantage of a
// transaction like processing of the results
func (pool *Pool) EnqueueTasks(CNs UList, proxy, certbase, hook string, format cert.Format) {
	go func() {
		for cn := range CNs {
			if req, err := cert.NewReq(cn, proxy, certbase, hook, format); err != nil {
				log.Fatal(err)
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

/*
func Worker(wid int, queue <-chan Task) {
TASK:
	// one task per domain
	for task := range queue {
		Verbose("[%d] Task %s\n", wid, task)
		env := []string{
			"DOMAIN=" + task.CN,
		}

		for _, req := range task.Requests {

			Verbose("[%d] Fetch %s\n", wid, req.URL)
			resp, err := http.DefaultClient.Do(req.Request)
			if err != nil {
				log.Fatal(err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				log.Printf("[%d] %s: Status %s\n", wid, req.URL, resp.Status)
				continue TASK
			}
			Verbose("[%d] %s: Status %s", wid, req.URL, resp.Status)

			var out io.WriteCloser
			if outfile := req.outfile; outfile == "-" {
				out = os.Stdout
			} else {
				dir, _ := filepath.Split(outfile)
				if err := Mkdir(dir); err != nil {
					log.Fatal(err)
				}
				out, err = os.Create(outfile)
				if err != nil {
					log.Fatal(err)
				}
				defer out.Close()
				Verbose("[%d] Writing to %s\n", wid, outfile)
			}
			if _, err := io.Copy(out, resp.Body); err != nil {
				log.Fatal(err)
			}

			var s string
			switch role := req.Role; role {
			case CERT:
				s = "CERTFILE="
			case CHAIN:
				s = "CHAINFILE="
			case PRIVKEY:
				s = "KEYFILE="
			case FULLCHAIN:
				s = "FULLCHAINFILE="
			case BUNDLE:
				s = "BUNDLEFILE="
			default:
				panic("Unknown certrole " + fmt.Sprint(role))
			}
			env = append(env, s+req.outfile)

		}

		hook := exec.Command(opt.Hook, "deploy_cert")
		hook.Env = env
		hook.Stdout = os.Stdout
		hook.Stderr = os.Stderr
		hook.Run()

	}
}
*/
