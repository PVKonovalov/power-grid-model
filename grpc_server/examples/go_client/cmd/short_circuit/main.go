// SPDX-FileCopyrightText: Contributors to the Power Grid Model project <powergridmodel@lfenergy.org>
//
// SPDX-License-Identifier: MPL-2.0

// Example Go client for power_grid_grpc_server's stateful model session API
// (CreateModel/UpdateAndCalculateShortCircuit/DestroyModel).
//
// This is a demonstration only and it is NOT intended to be used in production: the server address is
// hardcoded, there is no TLS/auth, and errors are handled by logging and exiting rather than being surfaced to
// a caller. It uses the same 110/35 kV substation feeder as power_grid_model_c_example/short_circuit_example.c,
// with a bolted three-phase fault placed on node_4. It only calculates once (with an empty
// update_dataset_json, i.e. no change since CreateModel) — see cmd/realtime_demo for repeated updates against
// the same model:
//
//	source_5 --node_1==(20 km 110 kV line_6)==node_2--[transformer_7: 110/35 kV, 25 MVA]--node_3==(10 km 35 kV
//	line_8)==node_4----sym_load_9
//	                                                                                          |
//	                                                                                       fault_10 (bolted three-phase)
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"power-grid-model/grpc_client_example/internal/samplenetworks"

	pgmpb "power-grid-model/grpc_client_example/internal/powergridpb"
)

// main dials power_grid_grpc_server, creates a model from the sample feeder-with-fault above (CreateModel),
// runs one short circuit calculation against it (UpdateAndCalculateShortCircuit, with no update — just
// recalculating what CreateModel built), prints the resulting sc_output dataset (always per-phase a/b/c) as
// pretty-printed JSON, then destroys the model (DestroyModel).
func main() {
	addr := flag.String("addr", "localhost:50051", "power_grid_grpc_server address")
	timeout := flag.Duration("timeout", 10*time.Second, "per-RPC timeout")
	flag.Parse()

	conn, err := grpc.NewClient(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to dial %s: %v", *addr, err)
	}
	defer conn.Close()

	client := pgmpb.NewPowerGridServiceClient(conn)
	ctx := context.Background()

	createCtx, cancel := context.WithTimeout(ctx, *timeout)
	createResp, err := client.CreateModel(createCtx, &pgmpb.CreateModelRequest{
		SystemFrequency:  50.0,
		InputDatasetJson: samplenetworks.FeederWithFault,
	})
	cancel()
	if err != nil {
		log.Fatalf("CreateModel failed: %v", err)
	}
	modelID := createResp.GetModelId()
	defer func() {
		destroyCtx, cancel := context.WithTimeout(context.Background(), *timeout)
		defer cancel()
		if _, err := client.DestroyModel(destroyCtx, &pgmpb.DestroyModelRequest{ModelId: modelID}); err != nil {
			log.Printf("DestroyModel failed: %v", err)
		}
	}()

	calcCtx, cancel := context.WithTimeout(ctx, *timeout)
	defer cancel()
	resp, err := client.UpdateAndCalculateShortCircuit(calcCtx, &pgmpb.UpdateAndCalculateShortCircuitRequest{
		ModelId: modelID,
		Options: &pgmpb.ShortCircuitOptions{
			VoltageScaling: pgmpb.ShortCircuitVoltageScaling_SHORT_CIRCUIT_VOLTAGE_SCALING_MAXIMUM,
		},
	})
	if err != nil {
		log.Fatalf("UpdateAndCalculateShortCircuit failed: %v", err)
	}

	var pretty map[string]any
	if err := json.Unmarshal([]byte(resp.GetOutputDatasetJson()), &pretty); err != nil {
		log.Fatalf("failed to parse output dataset JSON: %v", err)
	}
	out, err := json.MarshalIndent(pretty, "", "  ")
	if err != nil {
		log.Fatalf("failed to re-marshal output dataset JSON: %v", err)
	}
	fmt.Fprintln(os.Stdout, string(out))
}
