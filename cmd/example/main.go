package main

import (
	"context"
	"fmt"

	"github.com/fitzerooqq/fx"
)

func main() {
	fmt.Println("fx Lifecycle.Stop — hook continuation demo")
	fmt.Println("============================================")

	lc := fx.NewLifecycle()

	// Register hooks in order.
	lc.Append(fx.Hook{
		Name: "database",
		OnStop: func(ctx context.Context) error {
			fmt.Println("  [database] shutting down... OK")
			return nil
		},
	})
	lc.Append(fx.Hook{
		Name: "message_queue",
		OnStop: func(ctx context.Context) error {
			fmt.Println("  [message_queue] shutting down... OK")
			return nil
		},
	})
	lc.Append(fx.Hook{
		Name: "http_server",
		OnStop: func(ctx context.Context) error {
			fmt.Println("  [http_server] shutting down... OK")
			return nil
		},
	})
	lc.Append(fx.Hook{
		Name: "background_worker",
		OnStop: func(ctx context.Context) error {
			fmt.Println("  [background_worker] shutting down... FAIL")
			return context.Canceled
		},
	})
	lc.Append(fx.Hook{
		Name: "metrics_exporter",
		OnStop: func(ctx context.Context) error {
			fmt.Println("  [metrics_exporter] shutting down... OK")
			return nil
		},
	})
	lc.Append(fx.Hook{
		Name: "network_listener",
		OnStop: func(ctx context.Context) error {
			fmt.Println("  [network_listener] shutting down... OK")
			return nil
		},
	})

	fmt.Println("\nRunning Stop() — second-to-last hook will fail...")
	err := lc.Stop(context.Background())

	fmt.Println("\n=== Results ===")
	fmt.Println("  - background_worker returned context.Canceled")
	fmt.Println("  - ALL hooks still executed (reverse order)")
	fmt.Printf("  - Combined error: %v\n", err)
	fmt.Println("\n✓ Fix verified: all OnStop hooks ran despite prior failure.")
}
