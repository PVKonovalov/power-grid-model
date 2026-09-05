// SPDX-FileCopyrightText: Contributors to the Power Grid Model project <powergridmodel@lfenergy.org>
//
// SPDX-License-Identifier: MPL-2.0

// Example Go client for power_grid_grpc_server's stateful model session API
// (CreateModel/UpdateAndCalculateStateEstimation/DestroyModel).
//
// This is a demonstration only and it is NOT intended to be used in production: the server address is
// hardcoded, there is no TLS/auth, and errors are handled by logging and exiting rather than being surfaced to
// a caller. It estimates the state of the same single-node network as
// tests/data/state_estimation/single-node-source-sym-voltage-sensor/input.json: one node with a source and a
// single voltage sensor. With only one measurement and no other constraint, the weighted least squares estimate
// reproduces the sensor's measured voltage exactly (u = 12345.0 V, u_angle = 0.1 rad), matching that fixture's
// sym_output.json. It only calculates once (with an empty update_dataset_json, i.e. no change since
// CreateModel) — see cmd/realtime_demo for repeated updates against the same model.
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

// main dials power_grid_grpc_server, creates a model from the sample network above (CreateModel), runs one
// state estimation calculation against it (UpdateAndCalculateStateEstimation, with no update — just
// recalculating what CreateModel built), prints the estimated sym_output dataset as pretty-printed JSON, then
// destroys the model (DestroyModel).
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
		InputDatasetJson: samplenetworks.SingleNodeWithVoltageSensor,
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
	resp, err := client.UpdateAndCalculateStateEstimation(calcCtx, &pgmpb.UpdateAndCalculateStateEstimationRequest{
		ModelId: modelID,
		Options: &pgmpb.StateEstimationOptions{
			Symmetric:         true,
			CalculationMethod: pgmpb.CalculationMethod_CALCULATION_METHOD_ITERATIVE_LINEAR,
		},
	})
	if err != nil {
		log.Fatalf("UpdateAndCalculateStateEstimation failed: %v", err)
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
