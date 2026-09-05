// SPDX-FileCopyrightText: Contributors to the Power Grid Model project <powergridmodel@lfenergy.org>
//
// SPDX-License-Identifier: MPL-2.0

// Example Go client for power_grid_grpc_server's stateful model session API
// (CreateModel/UpdateAndCalculatePowerFlow/DestroyModel).
//
// This is a demonstration only and it is NOT intended to be used in production: the server address is
// hardcoded, there is no TLS/auth, and errors are handled by logging and exiting rather than being surfaced to
// a caller. It builds and solves the same small 110/35 kV substation feeder as
// power_grid_model_c_example/power_flow_example.c, by seeding it via the internal/pgmtypes structs (see
// buildFeeder below) rather than a hand-written JSON string. It only calculates once (with an empty
// update_dataset_json, i.e. no change since CreateModel) — see cmd/realtime_demo for repeated updates against
// the same model:
//
//	source_5 --node_1==(20 km 110 kV line_6)==node_2--[transformer_7: 110/35 kV, 25 MVA]--node_3==(10 km 35 kV
//	line_8)==node_4----sym_load_9
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

	"power-grid-model/grpc_client_example/internal/pgmtypes"

	pgmpb "power-grid-model/grpc_client_example/internal/powergridpb"
)

// buildFeeder returns the input dataset for the 110/35 kV substation feeder described above, as PGM input
// dataset JSON, built from the internal/pgmtypes structs instead of a hand-written JSON string. Field-for-field
// this is the same network as samplenetworks.Feeder (see that file for a raw-JSON version of the same grid).
func buildFeeder() (string, error) {
	dataset := pgmtypes.NewInputDataset(pgmtypes.InputData{
		Node: []pgmtypes.Node{
			{Base: pgmtypes.Base{ID: 1}, URated: 110e3},
			{Base: pgmtypes.Base{ID: 2}, URated: 110e3},
			{Base: pgmtypes.Base{ID: 3}, URated: 35e3},
			{Base: pgmtypes.Base{ID: 4}, URated: 35e3},
		},
		Line: []pgmtypes.Line{
			{
				BranchBase: pgmtypes.BranchBase{Base: pgmtypes.Base{ID: 6}, FromNode: 1, ToNode: 2, FromStatus: 1, ToStatus: 1},
				R1:         2.400, X1: 7.880, C1: 194e-9, Tan1: 0,
				R0: pgmtypes.Ptr(7.200), X0: pgmtypes.Ptr(23.600), C0: pgmtypes.Ptr(97e-9), Tan0: pgmtypes.Ptr(0.0),
				IN: pgmtypes.Ptr(500.0),
			},
			{
				BranchBase: pgmtypes.BranchBase{Base: pgmtypes.Base{ID: 8}, FromNode: 3, ToNode: 4, FromStatus: 1, ToStatus: 1},
				R1:         2.530, X1: 3.630, C1: 100e-9, Tan1: 0,
				R0: pgmtypes.Ptr(7.200), X0: pgmtypes.Ptr(11.500), C0: pgmtypes.Ptr(55e-9), Tan0: pgmtypes.Ptr(0.0),
				IN: pgmtypes.Ptr(350.0),
			},
		},
		Transformer: []pgmtypes.Transformer{
			{
				BranchBase: pgmtypes.BranchBase{Base: pgmtypes.Base{ID: 7}, FromNode: 2, ToNode: 3, FromStatus: 1, ToStatus: 1},
				U1:         110e3, U2: 35e3, Sn: 25e6, Uk: 0.105, Pk: 130e3, I0: 0.006, P0: 25e3,
				WindingFrom: pgmtypes.WindingWyeN, WindingTo: pgmtypes.WindingDelta, Clock: 11,
				TapSide: pgmtypes.FromSide, TapPos: pgmtypes.Ptr(int8(0)),
				TapMin: -8, TapMax: 8, TapNom: pgmtypes.Ptr(int8(0)), TapSize: 1375.0,
			},
		},
		Source: []pgmtypes.Source{
			{
				ApplianceBase: pgmtypes.ApplianceBase{Base: pgmtypes.Base{ID: 5}, Node: 1, Status: 1},
				URef:          pgmtypes.Ptr(1.0),
				Sk:            pgmtypes.Ptr(3000e6),
				RxRatio:       pgmtypes.Ptr(0.1),
			},
		},
		SymLoad: []pgmtypes.SymLoad{
			{
				ApplianceBase: pgmtypes.ApplianceBase{Base: pgmtypes.Base{ID: 9}, Node: 4, Status: 1},
				Type:          pgmtypes.ConstPower,
				PSpecified:    pgmtypes.Ptr(10e6),
				QSpecified:    pgmtypes.Ptr(3.287e6),
			},
		},
	})

	raw, err := json.Marshal(dataset)
	return string(raw), err
}

// main dials power_grid_grpc_server, creates a model from the sample feeder above (CreateModel), runs one power
// flow calculation against it (UpdateAndCalculatePowerFlow, with no update — just recalculating what
// CreateModel built), prints the resulting sym_output dataset as pretty-printed JSON, then destroys the model
// (DestroyModel).
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

	inputDatasetJSON, err := buildFeeder()
	if err != nil {
		log.Fatalf("failed to build input dataset: %v", err)
	}

	createCtx, cancel := context.WithTimeout(ctx, *timeout)
	createResp, err := client.CreateModel(createCtx, &pgmpb.CreateModelRequest{
		SystemFrequency:  50.0,
		InputDatasetJson: inputDatasetJSON,
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
	resp, err := client.UpdateAndCalculatePowerFlow(calcCtx, &pgmpb.UpdateAndCalculatePowerFlowRequest{
		ModelId: modelID,
		Options: &pgmpb.PowerFlowOptions{
			Symmetric:         true,
			CalculationMethod: pgmpb.CalculationMethod_CALCULATION_METHOD_NEWTON_RAPHSON,
		},
	})
	if err != nil {
		log.Fatalf("UpdateAndCalculatePowerFlow failed: %v", err)
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
