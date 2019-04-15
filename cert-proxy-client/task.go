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
	Name     string
	Requests []*http.Request
	Outfiles []string
}

func (task *Task) String() string {
	return fmt.Sprintf("%s (%d jobs)", task.Name, len(task.Requests))
}

// EnqueueTasks creates a task per CN, with all the items that are
// necessary for this CN. Bundling it per Domain has the advantage of a
// transaction like processing of the results
func enqueTasks(tasks chan<- Task, CNs UList, items []string) {
	for cn, _ := range CNs {
		var task = Task{Name: cn}
		for _, item := range items {
			req, err := http.NewRequest(`GET`, opt.Connect+`/`+path.Join(item, cn), nil)
			if err != nil {
				panic(err)
			}
			task.Requests = append(task.Requests, req)

			var outfile = opt.Outfile
			if outfile == "" {
				outfile = filepath.Join(opt.Certbase, cn, item+`.pem`)
			}
			task.Outfiles = append(task.Outfiles, outfile)

			verbose("Enqueing: %s\n", req.URL.String())
		}
		tasks <- task
	}
}

func Worker(wid int, queue <-chan Task) {
TASK:
	for task := range queue {
		verbose("[%d] Task %s\n", wid, task.String())

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
				log.Printf("[%d] Status %s: %s\n", wid, resp.Status, req.URL.String())
				continue TASK
			}
			verbose("[%d] Status %s: %s\n", wid, resp.Status, req.URL.String())

			var out io.WriteCloser
			if outfile := task.Outfiles[i]; outfile == "-" {
				out = os.Stdout
			} else {
				dir, _ := filepath.Split(outfile)
				if err := mkdir(dir); err != nil {
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

func mkdir(dir string) error {

	err := os.Mkdir(dir, 0777)

	if err != nil && os.IsExist(err) {
		stat, err := os.Stat(dir)
		if err != nil {
			return err
		}
		if stat.IsDir() {
			return nil
		}
	}
	return err
}
