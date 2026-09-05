<!--
SPDX-FileCopyrightText: Contributors to the Power Grid Model project <powergridmodel@lfenergy.org>

SPDX-License-Identifier: MPL-2.0
-->

# Go client examples for `power_grid_grpc_server`

Demonstrates calling `power_grid_grpc_server` (see `../../README.md`) from Go, via its stateful model session
API (`CreateModel` / `UpdateAndCalculate*` / `DestroyModel` — see `../../README.md`'s "Stateful model sessions"
section). One Go module, one example program per calculation type under `cmd/`, all sharing the same generated
stubs:

* `cmd/power_flow` — creates a model from the 110/35 kV substation feeder in
  `power_grid_model_c_example/power_flow_example.c` (built via `internal/pgmtypes`, see below, rather than a raw
  JSON string), runs one `UpdateAndCalculatePowerFlow` against it (no update — just recalculating what
  `CreateModel` built), and destroys the model.
* `cmd/state_estimation` — as `power_flow`, but `UpdateAndCalculateStateEstimation` on the single-node network
  from `tests/data/state_estimation/single-node-source-sym-voltage-sensor/input.json`.
* `cmd/short_circuit` — as `power_flow`, but `UpdateAndCalculateShortCircuit` on the same feeder as `power_flow`
  with a bolted three-phase fault added, from `power_grid_model_c_example/short_circuit_example.c`.
* `cmd/realtime_demo` — the session API's actual point: creates a model once, then repeatedly pushes a
  sinusoidally varying `sym_load_9` active power through `UpdateAndCalculatePowerFlow` and prints the resulting
  node 4 voltage, simulating a real-time SCADA-style feed. Run: `go run ./cmd/realtime_demo -iterations 20
  -interval 500ms`.
* `cmd/soak_test` — a load generator: creates the three models above once, then runs a configurable number of
  concurrent workers repeatedly calling `UpdateAndCalculatePowerFlow`/`UpdateAndCalculateStateEstimation`/
  `UpdateAndCalculateShortCircuit` (round-robin, no actual update each time) against them for a fixed duration,
  before destroying the models — exercising the session API's sustained-use path specifically. Useful for
  checking the server's memory stays flat under sustained load (see `../../README.md`'s notes on 24/7
  readiness). Run: `go run ./cmd/soak_test -duration 5m -concurrency 8`.

Packages under `internal/`:

* `powergridpb/` — the generated gRPC/protobuf stubs (`power_grid.pb.go`, `power_grid_grpc.pb.go`); regenerate
  after changing `../../proto/power_grid.proto` (see below).
* `pgmtypes/` — typed Go structs mirroring PGM's input dataset JSON schema, for building a grid by populating Go
  structs instead of hand-writing JSON. See `../../README.md`'s "Seeding input data from Go" and "Input dataset
  field reference" sections for the full field-by-field documentation.
* `samplenetworks/` — the same sample networks as raw JSON string constants (used by `state_estimation`,
  `short_circuit`, and `soak_test`; `power_flow` uses `pgmtypes` instead, as a worked example of the two
  approaches producing identical input).

`internal/powergridpb/` holds the generated stubs (`power_grid.pb.go`, `power_grid_grpc.pb.go`) — regenerate them
after changing `../../proto/power_grid.proto`:

```shell
protoc \
  -I ../../proto \
  --go_out=. --go_opt=module=power-grid-model/grpc_client_example \
  --go-grpc_out=. --go-grpc_opt=module=power-grid-model/grpc_client_example \
  ../../proto/power_grid.proto
```

(requires `protoc-gen-go`/`protoc-gen-go-grpc`, e.g. `go install google.golang.org/protobuf/cmd/protoc-gen-go@latest`
and `go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest`).

## Run

Start `power_grid_grpc_server` first (see `../../README.md`), then run any of the examples:

```shell
go run ./cmd/power_flow -addr localhost:50051
go run ./cmd/state_estimation -addr localhost:50051
go run ./cmd/short_circuit -addr localhost:50051
go run ./cmd/realtime_demo -addr localhost:50051
```

`power_flow`/`state_estimation`/`short_circuit` each print their calculation's output dataset
(`sym_output`/`asym_output`, or `sc_output` for short circuit) as pretty-printed JSON. `realtime_demo` instead
prints one line per update-and-recalculate iteration (load value pushed and the resulting node voltage).
