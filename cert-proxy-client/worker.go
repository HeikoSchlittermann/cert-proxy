package main

import (
	. "cert-proxy/internal/shared"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
)

// Task is a bundle of requests and associated destination
// files. Typically this is the bundle of operations necessary for
// one DN
type Task struct {
	CN       string
	Requests []*http.Request
	Outfiles []string
}

func (task Task) String() string {
	return fmt.Sprintf("%s (%d jobs)", task.CN, len(task.Requests))
}

// EnqueueTasks creates a task per CN, with all the items that are
// necessary for this CN. Bundling it per Domain has the advantage of a
// transaction like processing of the results
func enqueTasks(tasks chan<- Task, CNs UList, format Format, items []string) {
	for cn, _ := range CNs {
		var task = Task{CN: cn}
		for _, item := range items {

			req, err := http.NewRequest(`GET`, opt.Connect+path.Join(`/` + API_VERSION, item, cn), nil)
			if err != nil {
				panic(err)
			}
			req.URL.RawQuery = "format=" + format.String()

			outfile := opt.Outfile
			if outfile == "" {
				outfile = filepath.Join(opt.Certbase, cn, item+format.Ext())
			}

			task.Outfiles = append(task.Outfiles, outfile)
			task.Requests = append(task.Requests, req)
			verbose("Enqueing: %s ⇒ %s", req.URL.String(), outfile)
		}
		tasks <- task
	}
}

func Worker(wid int, queue <-chan Task) {
TASK:
	for task := range queue {
		verbose("[%d] Task %s\n", wid, task)

		if len(task.Requests) != len(task.Outfiles) {
			panic("number of requests != outfiles")
		}

		for i, req := range task.Requests {

			verbose("[%d] Fetch %s\n", wid, req.URL.String())
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				log.Fatal(err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				log.Printf("[%d] %s: Status %s\n", wid, req.URL, resp.Status)
				continue TASK
			}
			verbose("[%d] %s: Status %s", wid, req.URL, resp.Status)

			var out io.WriteCloser
			if outfile := task.Outfiles[i]; outfile == "-" {
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
				verbose("[%d] Writing to %s\n", wid, outfile)
			}
			if _, err := io.Copy(out, resp.Body); err != nil {
				log.Fatal(err)
			}
		}
	}
}
