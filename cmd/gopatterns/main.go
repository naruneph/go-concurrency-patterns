package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/naruneph/go-concurrency-patterns/registry"

	_ "github.com/naruneph/go-concurrency-patterns/primitives/cond"
	_ "github.com/naruneph/go-concurrency-patterns/primitives/mutex"
	_ "github.com/naruneph/go-concurrency-patterns/primitives/once"
	_ "github.com/naruneph/go-concurrency-patterns/primitives/waitgroup"
	_ "github.com/naruneph/go-concurrency-patterns/primitives/atomicx"
	_ "github.com/naruneph/go-concurrency-patterns/primitives/pool"

	_ "github.com/naruneph/go-concurrency-patterns/flows/selectloop"
	_ "github.com/naruneph/go-concurrency-patterns/flows/generator"
	_ "github.com/naruneph/go-concurrency-patterns/flows/pipeline"
	_ "github.com/naruneph/go-concurrency-patterns/flows/fanoutin"
	_ "github.com/naruneph/go-concurrency-patterns/flows/tee"
	_ "github.com/naruneph/go-concurrency-patterns/flows/or"
	_ "github.com/naruneph/go-concurrency-patterns/flows/ordone"
	_ "github.com/naruneph/go-concurrency-patterns/flows/bridge"

	_ "github.com/naruneph/go-concurrency-patterns/pools/semaphore"
	_ "github.com/naruneph/go-concurrency-patterns/pools/workerpool"
	_ "github.com/naruneph/go-concurrency-patterns/pools/workerpooldyn"

	_ "github.com/naruneph/go-concurrency-patterns/orchestration/errgroupx"
	_ "github.com/naruneph/go-concurrency-patterns/orchestration/shutdown"
	_ "github.com/naruneph/go-concurrency-patterns/orchestration/contextio"

	_ "github.com/naruneph/go-concurrency-patterns/scale/ratelimit"
	_ "github.com/naruneph/go-concurrency-patterns/scale/heartbeat"
	_ "github.com/naruneph/go-concurrency-patterns/scale/healing"
)

func main() {
	list := flag.Bool("list", false, "list available examples")
	example := flag.String("example", "", "example to run")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\nOptions:\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr, "\nExamples:")
		for _, name := range registry.List() {
			if ex, ok := registry.Examples[name]; ok && ex.Doc != "" {
				fmt.Fprintf(os.Stderr, " - %s: %s\n", name, ex.Doc)
				continue
			}
			fmt.Fprintf(os.Stderr, " - %s\n", name)
		}
	}

	flag.Parse()

	if *list {
		fmt.Println("Available examples:")
		for _, name := range registry.List() {
			if ex, ok := registry.Examples[name]; ok {
				fmt.Printf(" - %s: %s\n", name, ex.Doc)
			}
		}
		return
	}

	if ex, ok := registry.Examples[*example]; ok {
		ex.Func()
	} else {
		fmt.Println("Unknown or missing example. Use -list to see options.")
		os.Exit(1)
	}
}
