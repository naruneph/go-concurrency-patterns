package contextio

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/dolmen-go/contextio"
	"github.com/naruneph/go-concurrency-patterns/registry"
)

func init() {
	registry.Register(
		"orchestration_contextio",
		"Demonstrates a writer that respects context cancellation.",
		ContextIOExample,
	)
}

func ContextIOExample() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	writer := contextio.NewWriter(ctx, os.Stdout)

	for i := 0; ; i++ {
		_, err := fmt.Fprintf(writer, "line %d\n", i)
		if err != nil {
			fmt.Println("stopped:", err)
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
}
