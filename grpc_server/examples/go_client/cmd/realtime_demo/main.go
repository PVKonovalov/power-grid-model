// SPDX-FileCopyrightText: Contributors to the Power Grid Model project <powergridmodel@lfenergy.org>
//
// SPDX-License-Identifier: MPL-2.0

// Example: "real-time" usage of power_grid_grpc_server's stateful model session API
// (CreateModel/UpdateAndCalculatePowerFlow/DestroyModel).
//
// This is a demonstration only and it is NOT intended to be used in production: the server address is
// hardcoded and errors are handled by logging and exiting rather than being surfaced to a caller. It
// demonstrates the pattern for pushing frequent parameter changes (e.g. a SCADA load feed, or a tap change)
// without resending the whole grid or rebuilding topology on every call: CreateModel once, then repeatedly
// UpdateAndCalculatePowerFlow with just the changed attribute(s), and DestroyModel when done. Uses the same
// 110/35 kV feeder as the power_flow example, varying sym_load_9's active power on a sine wave to simulate a
// fluctuating real-world load.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"power-grid-model/grpc_client_example/internal/samplenetworks"

	pgmpb "power-grid-model/grpc_client_example/internal/powergridpb"
)

// outputDataset is the small slice of the sym_output dataset JSON this example actually reads: each node's id
// and per-unit voltage magnitude.
type outputDataset struct {
	Data struct {
		Node []struct {
			ID  int     `json:"id"`
			UPu float64 `json:"u_pu"`
		} `json:"node"`
	} `json:"data"`
}

// createModel calls CreateModel with the sample feeder and returns the resulting model_id, exiting the process
// on failure.
func createModel(ctx context.Context, client pgmpb.PowerGridServiceClient, timeout time.Duration) string {
	rpcCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	resp, err := client.CreateModel(rpcCtx, &pgmpb.CreateModelRequest{
		SystemFrequency:  50.0,
		InputDatasetJson: samplenetworks.Feeder,
	})
	if err != nil {
		log.Fatalf("CreateModel failed: %v", err)
	}
	fmt.Println("created model_id:", resp.GetModelId())
	return resp.GetModelId()
}

// destroyModel calls DestroyModel for modelID, logging (but not failing) on error — intended to be deferred so
// the session is cleaned up however main() exits.
func destroyModel(ctx context.Context, client pgmpb.PowerGridServiceClient, timeout time.Duration, modelID string) {
	rpcCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if _, err := client.DestroyModel(rpcCtx, &pgmpb.DestroyModelRequest{ModelId: modelID}); err != nil {
		log.Printf("DestroyModel failed: %v", err)
		return
	}
	fmt.Println("destroyed model_id:", modelID)
}

// main creates a model from the sample feeder, then repeatedly pushes a changed sym_load_9 active power
// through UpdateAndCalculatePowerFlow (simulating a fluctuating real-time load reading) and prints the
// resulting node 4 voltage, before destroying the model.
func main() {
	addr := flag.String("addr", "localhost:50051", "power_grid_grpc_server address")
	iterations := flag.Int("iterations", 10, "number of update-and-recalculate iterations to run")
	interval := flag.Duration("interval", time.Second, "delay between iterations")
	timeout := flag.Duration("timeout", 10*time.Second, "per-RPC timeout")
	flag.Parse()

	conn, err := grpc.NewClient(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to dial %s: %v", *addr, err)
	}
	defer conn.Close()

	client := pgmpb.NewPowerGridServiceClient(conn)
	ctx := context.Background()

	modelID := createModel(ctx, client, *timeout)
	defer destroyModel(ctx, client, *timeout, modelID)

	const basePowerW = 10e6  // 10 MW baseline, matching the static power_flow example
	const swingPowerW = 4e6  // +/- 4 MW swing to simulate a fluctuating real load
	const powerFactor = 0.95 // matches the static example's q_specified/p_specified ratio

	for i := 0; i < *iterations; i++ {
		activePower := basePowerW + swingPowerW*math.Sin(float64(i)/2)
		reactivePower := activePower * math.Tan(math.Acos(powerFactor))

		// Only the changed attributes are sent — id to address the component, p_specified/q_specified to
		// change it. Everything else about the grid (topology, all other components) is left untouched.
		updateDatasetJSON := fmt.Sprintf(
			`{"version":"1.0","type":"update","is_batch":false,"attributes":{},`+
				`"data":{"sym_load":[{"id":9,"p_specified":%f,"q_specified":%f}]}}`,
			activePower, reactivePower)

		rpcCtx, cancel := context.WithTimeout(ctx, *timeout)
		resp, err := client.UpdateAndCalculatePowerFlow(rpcCtx, &pgmpb.UpdateAndCalculatePowerFlowRequest{
			ModelId:           modelID,
			UpdateDatasetJson: updateDatasetJSON,
			Options: &pgmpb.PowerFlowOptions{
				Symmetric:         true,
				CalculationMethod: pgmpb.CalculationMethod_CALCULATION_METHOD_NEWTON_RAPHSON,
			},
		})
		cancel()
		if err != nil {
			log.Fatalf("iteration %d: UpdateAndCalculatePowerFlow failed: %v", i, err)
		}

		var output outputDataset
		if err := json.Unmarshal([]byte(resp.GetOutputDatasetJson()), &output); err != nil {
			log.Fatalf("iteration %d: failed to parse output dataset JSON: %v", i, err)
		}

		fmt.Printf("iteration %2d: load p_specified=%8.0f W -> node %d u_pu=%.6f\n",
			i, activePower, output.Data.Node[3].ID, output.Data.Node[3].UPu)

		if i < *iterations-1 {
			time.Sleep(*interval)
		}
	}
}
