// SPDX-FileCopyrightText: Contributors to the Power Grid Model project <powergridmodel@lfenergy.org>
//
// SPDX-License-Identifier: MPL-2.0

// Soak test load generator for power_grid_grpc_server's stateful model session API.
//
// Creates three models once up front (CreateModel, one per calculation type/sample network), then runs a
// configurable number of concurrent workers, each repeatedly calling UpdateAndCalculatePowerFlow,
// UpdateAndCalculateStateEstimation, and UpdateAndCalculateShortCircuit in turn (round-robin, with no actual
// update each time — just recalculating) against those shared models, for a fixed duration, before destroying
// them (DestroyModel). This exercises the session API's sustained-use path specifically, since that's the one
// meant for high-frequency "real-time" use — see ../../README.md's "Stateful model sessions" section. It only
// exercises the client/network/RPC path — it does not measure the server process's memory itself; run it while
// sampling the server's RSS externally (e.g. `ps -o rss= -p <server_pid>` on a timer) to check whether memory
// stays flat under sustained load. See ../../README.md for an example driver script.
package main

import (
	"context"
	"flag"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"power-grid-model/grpc_client_example/internal/samplenetworks"

	pgmpb "power-grid-model/grpc_client_example/internal/powergridpb"
)

// modelIDs holds one model_id per calculation type, all created once up front and shared by every worker.
type modelIDs struct {
	powerFlow       string
	stateEstimation string
	shortCircuit    string
}

// createModels calls CreateModel once for each of the three sample networks, returning their model_ids. Exits
// the process on failure.
func createModels(ctx context.Context, client pgmpb.PowerGridServiceClient, timeout time.Duration) modelIDs {
	create := func(inputDatasetJSON string) string {
		rpcCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		resp, err := client.CreateModel(rpcCtx, &pgmpb.CreateModelRequest{
			SystemFrequency:  50.0,
			InputDatasetJson: inputDatasetJSON,
		})
		if err != nil {
			log.Fatalf("CreateModel failed: %v", err)
		}
		return resp.GetModelId()
	}

	return modelIDs{
		powerFlow:       create(samplenetworks.Feeder),
		stateEstimation: create(samplenetworks.SingleNodeWithVoltageSensor),
		shortCircuit:    create(samplenetworks.FeederWithFault),
	}
}

// destroyModels calls DestroyModel for each of ids's models, logging (but not failing) on error.
func destroyModels(ctx context.Context, client pgmpb.PowerGridServiceClient, timeout time.Duration, ids modelIDs) {
	destroy := func(modelID string) {
		rpcCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		if _, err := client.DestroyModel(rpcCtx, &pgmpb.DestroyModelRequest{ModelId: modelID}); err != nil {
			log.Printf("DestroyModel(%s) failed: %v", modelID, err)
		}
	}
	destroy(ids.powerFlow)
	destroy(ids.stateEstimation)
	destroy(ids.shortCircuit)
}

// runOneRPC issues a single UpdateAndCalculate* request (with an empty update_dataset_json, i.e. just
// recalculating), round-robining across the three calculation types/models by iteration count, and reports
// whether it succeeded.
func runOneRPC(ctx context.Context, client pgmpb.PowerGridServiceClient, ids modelIDs, iteration int) error {
	rpcCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	switch iteration % 3 {
	case 0:
		_, err := client.UpdateAndCalculatePowerFlow(rpcCtx, &pgmpb.UpdateAndCalculatePowerFlowRequest{
			ModelId: ids.powerFlow,
			Options: &pgmpb.PowerFlowOptions{
				Symmetric:         true,
				CalculationMethod: pgmpb.CalculationMethod_CALCULATION_METHOD_NEWTON_RAPHSON,
			},
		})
		return err
	case 1:
		_, err := client.UpdateAndCalculateStateEstimation(rpcCtx, &pgmpb.UpdateAndCalculateStateEstimationRequest{
			ModelId: ids.stateEstimation,
			Options: &pgmpb.StateEstimationOptions{
				Symmetric:         true,
				CalculationMethod: pgmpb.CalculationMethod_CALCULATION_METHOD_ITERATIVE_LINEAR,
			},
		})
		return err
	default:
		_, err := client.UpdateAndCalculateShortCircuit(rpcCtx, &pgmpb.UpdateAndCalculateShortCircuitRequest{
			ModelId: ids.shortCircuit,
			Options: &pgmpb.ShortCircuitOptions{
				VoltageScaling: pgmpb.ShortCircuitVoltageScaling_SHORT_CIRCUIT_VOLTAGE_SCALING_MAXIMUM,
			},
		})
		return err
	}
}

// worker runs runOneRPC in a tight loop, counting requests and errors into total/errors, until ctx is done.
func worker(ctx context.Context, client pgmpb.PowerGridServiceClient, ids modelIDs, total, errors *atomic.Int64) {
	for iteration := 0; ; iteration++ {
		select {
		case <-ctx.Done():
			return
		default:
		}
		if err := runOneRPC(ctx, client, ids, iteration); err != nil {
			errors.Add(1)
			log.Printf("request failed: %v", err)
		}
		total.Add(1)
	}
}

// main dials power_grid_grpc_server, creates the three shared models, runs -concurrency workers for -duration
// against them (logging a progress line every 10s and a final request/error count when done), then destroys
// the models.
func main() {
	addr := flag.String("addr", "localhost:50051", "power_grid_grpc_server address")
	duration := flag.Duration("duration", 5*time.Minute, "how long to run the soak test")
	concurrency := flag.Int("concurrency", 8, "number of concurrent goroutines issuing requests")
	modelTimeout := flag.Duration("model-timeout", 10*time.Second, "per-RPC timeout for CreateModel/DestroyModel")
	flag.Parse()

	conn, err := grpc.NewClient(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to dial %s: %v", *addr, err)
	}
	defer conn.Close()

	client := pgmpb.NewPowerGridServiceClient(conn)

	ids := createModels(context.Background(), client, *modelTimeout)
	defer destroyModels(context.Background(), client, *modelTimeout, ids)

	ctx, cancel := context.WithTimeout(context.Background(), *duration)
	defer cancel()

	var total, errors atomic.Int64

	var wg sync.WaitGroup
	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			worker(ctx, client, ids, &total, &errors)
		}()
	}

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	progressDone := make(chan struct{})
	go func() {
		defer close(progressDone)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				log.Printf("progress: %d requests, %d errors", total.Load(), errors.Load())
			}
		}
	}()

	wg.Wait()
	<-progressDone
	log.Printf("done: %d requests, %d errors over %s (concurrency=%d)", total.Load(), errors.Load(), *duration,
		*concurrency)
}
