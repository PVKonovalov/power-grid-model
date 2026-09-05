<!--
SPDX-FileCopyrightText: Contributors to the Power Grid Model project <powergridmodel@lfenergy.org>

SPDX-License-Identifier: MPL-2.0
-->

# `power_grid_grpc_server`

A gRPC service that links directly against the installed `power_grid_model_c` / `power_grid_model_cpp` package
and exposes power flow, state estimation, and short circuit calculations to non-C++ clients (e.g. a Go backend)
over the network, instead of binding to the C ABI directly (cgo, `dlopen`, etc.).

See `proto/power_grid.proto` for the service definition (`CalculatePowerFlow`, `CalculateStateEstimation`,
`CalculateShortCircuit`). Request/response payloads carry PGM's own input/output dataset JSON (the same schema
used by `power_grid_model_c_example/serialization.c`) as string fields.

Those three RPCs are stateless: each call parses a complete input dataset, builds a model, solves it, and
discards it. For "real-time" use — pushing frequent parameter changes (a SCADA load reading, a tap position)
and recalculating — see [Stateful model sessions](#stateful-model-sessions-real-time-updates) below, which keeps
a model alive across calls instead.

Every calculation (stateless or stateful) logs one line to stdout: which RPC ran, the grid's node/branch count,
and how long the actual solve took, e.g. `[calc] CalculatePowerFlow: nodes=4 branches=3 duration_ms=0.15`.

## Run with Docker

The easiest way to build and run this: `Dockerfile` builds the core library and the server from source in one
multi-stage build (no local toolchain/dependencies needed beyond Docker itself), producing a minimal runtime
image (Ubuntu base + just the runtime shared libraries, no compilers/headers) with a `HEALTHCHECK` that calls
the `grpc.health.v1.Health` service `main.cpp` already enables.

```shell
docker compose up --build
```

Listens on `localhost:50051`. Configurable via environment variables in `docker-compose.yml`: `GRPC_PORT`
(default `50051`) and `GRPC_SHUTDOWN_TIMEOUT_SECONDS` (default `10`, see `main.cpp`'s graceful-shutdown
handling). `restart: unless-stopped` pairs with the healthcheck so Docker restarts the container if it ever
crashes or stops responding.

To build the image without compose: `docker build -f grpc_server/Dockerfile -t power-grid-grpc-server ..` from
the repo root (note the `..` — the build context must be the repo root, not `grpc_server/`, since the image also
builds the core library from `../power_grid_model_c`).

## Build from source (native, e.g. for local development/debugging)

### Prerequisites

```shell
brew install grpc protobuf
export CMAKE_PREFIX_PATH=/opt/homebrew
```

If `grpc`/`protobuf` were already installed and you upgrade them with `brew upgrade grpc protobuf`, also
upgrade `re2` (`brew upgrade re2`): Homebrew's `grpc` bottle depends on `re2`, which itself links against
`abseil`. Bumping `protobuf`/`grpc` can bump `abseil`'s SONAME along with it, leaving an already-installed `re2`
dynamically linked against an `abseil` version that no longer exists on disk. Symptom is a `dyld` "Library not
loaded: .../libabsl_*.dylib" error when starting the server. If that ever recurs, `brew upgrade` (or reinstall)
whichever formula's error message names.

### Build

The core library must be built and installed first (from the repo root):

```shell
./build.sh -p apple-clang-release -i
```

Then, from this directory:

```shell
cmake --preset apple-clang-release
cmake --build --preset apple-clang-release --verbose -j1
```

### Run

```shell
DYLD_LIBRARY_PATH=../install/apple-clang-release/lib:/opt/homebrew/lib \
  ./build/apple-clang-release/power_grid_grpc_server
```

Listens on `:50051` by default (override with the `GRPC_PORT` env var; `GRPC_SHUTDOWN_TIMEOUT_SECONDS` controls
how long SIGINT/SIGTERM waits for in-flight RPCs to drain before forcing shutdown, default `10`).

## Try it

Server reflection is enabled, so `grpcurl` works without a compiled client:

```shell
brew install grpcurl
grpcurl -plaintext -d '{
  "system_frequency": 50.0,
  "input_dataset_json": "{\"version\":\"1.0\",\"type\":\"input\",\"is_batch\":false,\"attributes\":{},\"data\":{\"node\":[{\"id\":0,\"u_rated\":10e3}],\"source\":[{\"id\":1,\"node\":0,\"status\":1,\"u_ref\":1,\"sk\":1e20}],\"sym_load\":[{\"id\":2,\"node\":0,\"status\":1,\"type\":0,\"p_specified\":0,\"q_specified\":0}]}}",
  "options": {"symmetric": true, "calculation_method": "CALCULATION_METHOD_NEWTON_RAPHSON"}
}' localhost:50051 power_grid_model.v1.PowerGridService/CalculatePowerFlow
```

Or run the Go client examples in `examples/go_client/` (one per RPC, including state estimation and short
circuit) — see `examples/go_client/README.md`.

## Stateful model sessions (real-time updates)

`CreateModel` / `UpdateAndCalculatePowerFlow` / `UpdateAndCalculateStateEstimation` /
`UpdateAndCalculateShortCircuit` / `DestroyModel` keep a model alive in server memory across calls, instead of
rebuilding it from scratch every time. This matters for "real-time" use: pushing a changed load reading or tap
position and recalculating only needs to touch the changed values — PGM's underlying `PGM_update_model` skips
topology/Y-bus reconstruction entirely unless a topology-affecting attribute (e.g. `from_status`/`to_status`)
actually changed, which it detects automatically.

```shell
# 1. Create a model once, get back a model_id
MODEL_ID=$(grpcurl -plaintext -d '{
  "system_frequency": 50.0,
  "input_dataset_json": "{\"version\":\"1.0\",\"type\":\"input\",\"is_batch\":false,\"attributes\":{},\"data\":{\"node\":[{\"id\":0,\"u_rated\":10e3}],\"source\":[{\"id\":1,\"node\":0,\"status\":1,\"u_ref\":1,\"sk\":1e20}],\"sym_load\":[{\"id\":2,\"node\":0,\"status\":1,\"type\":0,\"p_specified\":0,\"q_specified\":0}]}}"
}' localhost:50051 power_grid_model.v1.PowerGridService/CreateModel | jq -r .model_id)

# 2. Recalculate as often as you like, pushing only the changed attributes each time (id + changed fields;
#    everything else about the grid is left untouched). update_dataset_json may be empty to just recalculate.
grpcurl -plaintext -d '{
  "model_id": "'"$MODEL_ID"'",
  "update_dataset_json": "{\"version\":\"1.0\",\"type\":\"update\",\"is_batch\":false,\"attributes\":{},\"data\":{\"sym_load\":[{\"id\":2,\"p_specified\":5e6,\"q_specified\":1.5e6}]}}",
  "options": {"symmetric": true, "calculation_method": "CALCULATION_METHOD_NEWTON_RAPHSON"}
}' localhost:50051 power_grid_model.v1.PowerGridService/UpdateAndCalculatePowerFlow

# 3. Clean up when the session is done
grpcurl -plaintext -d '{"model_id": "'"$MODEL_ID"'"}' localhost:50051 power_grid_model.v1.PowerGridService/DestroyModel
```

Or run `examples/go_client/cmd/realtime_demo` for a worked example (creates a model, then loops pushing a
sinusoidally varying load and printing the resulting voltage, before destroying the model).

A few things worth knowing before relying on this in production:

* **`model_id` is a server-memory-only key.** It's a simple incrementing counter — not authenticated, and it
  does not survive a server restart. Anyone who can reach the server can use any `model_id`; put this behind
  your own auth/network boundary if that matters.
* **Models are never garbage-collected.** There is no expiry and no idle-session reaper — always call
  `DestroyModel` when a session ends, or its memory (and the grid's node/branch/attribute buffers) leaks for the
  life of the process.
* **Calls against the same `model_id` are serialized**, not run concurrently (the server holds a per-model
  mutex for the duration of each `UpdateAndCalculate*` call) — safe, but a slow calculation on one `model_id`
  will queue up other calls against that *same* `model_id`. Different `model_id`s run fully in parallel.

## Seeding input data from Go

`examples/go_client/internal/pgmtypes` provides typed Go structs mirroring every field in PGM's input dataset
JSON schema, so you can build a grid by populating Go structs instead of hand-writing JSON:

```go
dataset := pgmtypes.NewInputDataset(pgmtypes.InputData{
    Node: []pgmtypes.Node{
        {Base: pgmtypes.Base{ID: 1}, URated: 10e3},
    },
    Source: []pgmtypes.Source{
        {
            ApplianceBase: pgmtypes.ApplianceBase{Base: pgmtypes.Base{ID: 2}, Node: 1, Status: 1},
            URef:          pgmtypes.Ptr(1.0),
        },
    },
    SymLoad: []pgmtypes.SymLoad{
        {
            ApplianceBase: pgmtypes.ApplianceBase{Base: pgmtypes.Base{ID: 3}, Node: 1, Status: 1},
            Type:          pgmtypes.ConstPower,
            PSpecified:    pgmtypes.Ptr(0.0),
            QSpecified:    pgmtypes.Ptr(0.0),
        },
    },
})

inputDatasetJSON, err := json.Marshal(dataset) // -> use as CalculatePowerFlowRequest.InputDatasetJson
```

See `cmd/power_flow/main.go`'s `buildFeeder` function for a full worked example (the 110/35 kV feeder), verified
to produce byte-for-byte the same JSON as the hand-written version in `internal/samplenetworks`.

A few design notes that matter when using the package:

* **Optional attributes are pointer fields** (`*float64`, `*int8`, `*WindingType`, …) with `json:",omitempty"`,
  so an unset field is genuinely *absent* from the JSON — letting PGM apply its own default — rather than being
  sent as a zero value (which, for many attributes, is a real and different value from "not specified"). Use the
  generic `pgmtypes.Ptr(v)` helper to populate them inline, e.g. `pgmtypes.Ptr(0.0)`. Required fields (no
  documented default, always needed) are plain values.
* **Every component embeds a shared base struct** (`Base`, `BranchBase`, `Branch3Side`, `ApplianceBase`,
  `SensorBase`, `RegulatorBase`) matching PGM's own component inheritance — Go's struct embedding flattens these
  into the same JSON object, so e.g. `Line{BranchBase: pgmtypes.BranchBase{...}, R1: ...}` marshals with
  `from_node`/`to_node`/`from_status`/`to_status`/`id` as siblings of `r1`, not nested.
* **Symmetric/asymmetric variants share one generic struct.** `LoadGen[P]`, `VoltageSensor[P]`, `PowerSensor[P]`,
  `CurrentSensor[P]` are generic over `P` (`float64` for the symmetric variant, `[3]float64` for the per-phase
  a/b/c asymmetric variant); `SymLoad`, `AsymLoad`, `SymVoltageSensor`, `AsymVoltageSensor`, etc. are type
  aliases for a concrete instantiation. This mirrors how PGM's own C++ core and JSON schema model them — see
  the field reference below.
* Regenerate nothing here when `proto/power_grid.proto` changes — `pgmtypes` is independent of the generated
  gRPC stubs; it only concerns the `input_dataset_json`/`output_dataset_json` string payloads.

## Input dataset field reference

Full attribute reference for every component type PGM's input dataset supports, adapted from
`docs/user_manual/components.md` in the repo root (the authoritative source — consult it for the underlying
electrical models, validation-rule tables, and worked examples omitted here for brevity). Columns:

* **JSON field** — the key as it appears in `input_dataset_json`.
* **Go field** (`pgmtypes.<Type>.<Field>`) — `—` for fields inherited from an embedded base struct (see the
  component's embedded base type, named in its heading).
* **Type** — the JSON field's data type; `int8`/`int32`/`float64` are plain JSON numbers, `[3]float64` is a
  3-element JSON array (one value per phase a/b/c), and named types (e.g. `WindingType`) are `int8` enums (see
  [Enum reference](#enum-reference)).
* **Required** — whether the field must be present; many are conditionally required (e.g. "only for asymmetric
  calculations") — see the description.

### Base fields (every component)

Go: `pgmtypes.Base` (embedded in every component type below).

| JSON field | Go field | Type    | Required | Description                                                                          |
|------------|----------|---------|:--------:|---------------------------------------------------------------------------------------|
| `id`       | `ID`     | `int32` |   yes    | ID of this component; must be unique across all components within the same scenario. |

### Node

Type name: `node`. Go: `pgmtypes.Node` (embeds `Base`).

| JSON field | Go field | Type      | Unit        | Required | Description                          |
|------------|----------|-----------|-------------|:--------:|---------------------------------------|
| `u_rated`  | `URated` | `float64` | volt (V)    |   yes    | Rated line-line voltage. Must be > 0. |

### Line

A branch with specified serial impedance and shunt admittance; connects two nodes at the *same* rated voltage.
Type name: `line`. Go: `pgmtypes.Line` (embeds `BranchBase`: `FromNode`, `ToNode`, `FromStatus`, `ToStatus` — all
`int32`/`int8`, all required).

| JSON field | Go field | Type      | Unit      | Required                         | Description                                          |
|------------|----------|-----------|-----------|:---------------------------------:|-------------------------------------------------------|
| `r1`       | `R1`     | `float64` | ohm (Ω)   | yes                               | Positive-sequence serial resistance. `r1`/`x1` cannot both be 0. |
| `x1`       | `X1`     | `float64` | ohm (Ω)   | yes                               | Positive-sequence serial reactance.                   |
| `c1`       | `C1`     | `float64` | farad (F) | yes                               | Positive-sequence shunt capacitance.                  |
| `tan1`     | `Tan1`   | `float64` | -         | yes                               | Positive-sequence shunt loss factor (tan δ).          |
| `r0`       | `R0`     | `float64` | ohm (Ω)   | only for asymmetric calculations | Zero-sequence serial resistance. `r0`/`x0` cannot both be 0. |
| `x0`       | `X0`     | `float64` | ohm (Ω)   | only for asymmetric calculations | Zero-sequence serial reactance.                       |
| `c0`       | `C0`     | `float64` | farad (F) | only for asymmetric calculations | Zero-sequence shunt capacitance.                      |
| `tan0`     | `Tan0`   | `float64` | -         | only for asymmetric calculations | Zero-sequence shunt loss factor (tan δ).              |
| `i_n`      | `IN`     | `float64` | ampere (A)| no                                | Rated current, must be > 0. If omitted, `loading` in the output is NaN. |

### Asym Line

A branch with resistance/reactance/capacitance given per phase (a/b/c, optionally n for neutral) instead of as
symmetric-component parameters; connects two nodes at the same rated voltage. Type name: `asym_line`. Go:
`pgmtypes.AsymLine` (embeds `BranchBase`). See `docs/user_manual/components.md#asym-line` for the exact
required-field combinations (full r/x matrix with/without neutral; c-matrix vs `c0`+`c1`) — summarized here.

| JSON field | Go field | Type | Unit | Required | Description |
| -------------------------------------------- | ---------------------------------- | ----------- | ------------ | ----------------------------------------------- | -------------- |
| `r_aa`,`r_ba`,`r_bb`,`r_ca`,`r_cb`,`r_cc` | `RAA`,`RBA`,`RBB`,`RCA`,`RCB`,`RCC` | `float64` | ohm (Ω) | yes | Series resistance matrix entries (aa/bb/cc must be > 0, others >= 0). |
| `r_na`,`r_nb`,`r_nc`,`r_nn` | `RNA`,`RNB`,`RNC`,`RNN` | `float64` | ohm (Ω) | all-or-none, for a 4th (neutral) phase | Neutral-phase series resistance matrix entries. |
| `x_aa` … `x_cc` | `XAA` … `XCC` | `float64` | ohm (Ω) | yes | Series reactance matrix entries (same shape as r). |
| `x_na`,`x_nb`,`x_nc`,`x_nn` | `XNA`,`XNB`,`XNC`,`XNN` | `float64` | ohm (Ω) | all-or-none, for a 4th (neutral) phase | Neutral-phase series reactance matrix entries. |
| `c_aa`,`c_ba`,`c_bb`,`c_ca`,`c_cb`,`c_cc` | `CAA`,`CBA`,`CBB`,`CCA`,`CCB`,`CCC` | `float64` | farad (F) | full matrix, or use `c0`+`c1` instead | Shunt nodal capacitance matrix entries. |
| `c0`, `c1` | `C0`, `C1` | `float64` | farad (F) | alternative to the full c matrix | Zero-/positive-sequence shunt capacitance (used in preference if both given). |
| `i_n` | `IN` | `float64` | ampere (A) | no | Rated current, must be > 0. If omitted, `loading` is NaN. |

### Link

A branch with a fixed, very high admittance (effectively zero impedance) — no additional attributes beyond the
edge base fields. No sensors can be coupled to a `link`. Type name: `link`. Go: `pgmtypes.Link` (embeds
`BranchBase` only).

### Generic Branch

A branch parameterized directly by PI-model electrical parameters rather than transformer-style ratings; can
behave like a line or transformer depending on the chosen parameters. Asymmetric calculation is **not**
supported. Type name: `generic_branch`. Go: `pgmtypes.GenericBranch` (embeds `BranchBase`).

| JSON field | Go field | Type      | Unit             | Required | Description                                                        |
|------------|----------|-----------|------------------|:--------:|----------------------------------------------------------------------|
| `r1`       | `R1`     | `float64` | ohm              |   yes    | Positive-sequence resistance, referenced to the "to" side.          |
| `x1`       | `X1`     | `float64` | ohm              |   yes    | Positive-sequence reactance, referenced to the "to" side.           |
| `g1`       | `G1`     | `float64` | siemens          |   yes    | Positive-sequence conductance, referenced to the "to" side.         |
| `b1`       | `B1`     | `float64` | siemens          |   yes    | Positive-sequence susceptance, referenced to the "to" side.         |
| `k`        | `K`      | `float64` | -                |    no    | Off-nominal ratio (not the nominal voltage ratio). Default `1.0`. Must be > 0. |
| `theta`    | `Theta`  | `float64` | radian           |    no    | Angle shift. Default `0.0`.                                          |
| `sn`       | `Sn`     | `float64` | volt-ampere (VA) |    no    | Rated power, used only for loading output. Default `0.0`. Must be >= 0. |

### Transformer

A branch connecting two nodes at possibly different voltage levels. Type name: `transformer`. Go:
`pgmtypes.Transformer` (embeds `BranchBase`).

| JSON field | Go field | Type | Unit | Required | Description |
| ----------------------- | --------------------- | --------------- | ------------------ | ------------------------------------------------ | -------------- |
| `u1` | `U1` | `float64` | volt (V) | yes | Rated voltage at from-side. Must be > 0. |
| `u2` | `U2` | `float64` | volt (V) | yes | Rated voltage at to-side. Must be > 0. |
| `sn` | `Sn` | `float64` | volt-ampere (VA) | yes | Rated power. Must be > 0. |
| `uk` | `Uk` | `float64` | - | yes | Relative short circuit voltage (`0.1` = 10%). Must be >= `pk/sn`, > 0, < 1. |
| `pk` | `Pk` | `float64` | watt (W) | yes | Short circuit (copper) loss. Must be >= 0. |
| `i0` | `I0` | `float64` | - | yes | Relative no-load (magnetizing) current. Must be >= `p0/sn`, < 1. |
| `p0` | `P0` | `float64` | watt (W) | yes | No-load (iron/magnetizing) loss. Must be >= 0. |
| `i0_zero_sequence` | `I0ZeroSequence` | `float64` | - | no, default `i0` | Zero-sequence relative no-load current. |
| `p0_zero_sequence` | `P0ZeroSequence` | `float64` | watt (W) | no, default `p0 + pk*(i0_zero_sequence²-i0²)` | Zero-sequence no-load loss. |
| `winding_from` | `WindingFrom` | `WindingType` | - | yes | From-side winding type. |
| `winding_to` | `WindingTo` | `WindingType` | - | yes | To-side winding type. |
| `clock` | `Clock` | `int8` | - | yes | Clock number of phase shift, `-12..12` (even invalid if exactly one side is Y(N); odd invalid if both/neither side is Y(N)). |
| `tap_side` | `TapSide` | `BranchSide` | - | yes | Side of the tap changer. |
| `tap_pos` | `TapPos` | `int8` | - | no, default `tap_nom` (or `0`) | Current tap position; must be between `tap_min`/`tap_max`. |
| `tap_min` | `TapMin` | `int8` | - | yes | Tap position at minimum voltage (may be > `tap_max`). |
| `tap_max` | `TapMax` | `int8` | - | yes | Tap position at maximum voltage. |
| `tap_nom` | `TapNom` | `int8` | - | no, default `0` | Nominal tap position; must be between `tap_min`/`tap_max`. |
| `tap_size` | `TapSize` | `float64` | volt (V) | yes | Voltage size of each tap. Must be >= 0. |
| `uk_min`, `uk_max` | `UkMin`, `UkMax` | `float64` | - | no, default `uk` | Relative short circuit voltage at minimum/maximum tap. |
| `pk_min`, `pk_max` | `PkMin`, `PkMax` | `float64` | watt (W) | no, default `pk` | Short circuit loss at minimum/maximum tap. |
| `r_grounding_from`, `x_grounding_from` | `RGroundingFrom`, `XGroundingFrom` | `float64` | ohm (Ω) | no, default `0` | Grounding resistance/reactance at from-side, if relevant. |
| `r_grounding_to`, `x_grounding_to` | `RGroundingTo`, `XGroundingTo` | `float64` | ohm (Ω) | no, default `0` | Grounding resistance/reactance at to-side, if relevant. |

### Three-Winding Transformer

A branch3 connecting three nodes at possibly different voltage levels. Type name: `three_winding_transformer`.
Go: `pgmtypes.ThreeWindingTransformer` (embeds `Branch3Base`: `Node1`/`Node2`/`Node3`, `Status1`/`Status2`/`Status3`
— all required).

| JSON field | Go field | Type | Unit | Required | Description |
| ----------------------------- | -------------------------------- | ---------------- | ------------------ | -------------------------------------- | -------------- |
| `u1`,`u2`,`u3` | `U1`,`U2`,`U3` | `float64` | volt (V) | yes | Rated voltage at side 1/2/3. Must be > 0. |
| `sn_1`,`sn_2`,`sn_3` | `Sn1`,`Sn2`,`Sn3` | `float64` | volt-ampere (VA) | yes | Rated power at side 1/2/3. Must be > 0. |
| `uk_12`,`uk_13`,`uk_23` | `Uk12`,`Uk13`,`Uk23` | `float64` | - | yes | Relative short circuit voltage across side pairs. |
| `pk_12`,`pk_13`,`pk_23` | `Pk12`,`Pk13`,`Pk23` | `float64` | watt (W) | yes | Short circuit loss across side pairs. Must be >= 0. |
| `i0` | `I0` | `float64` | - | yes | Relative no-load current w.r.t. side 1. |
| `p0` | `P0` | `float64` | watt (W) | yes | No-load loss. Must be >= 0. |
| `winding_1`,`winding_2`,`winding_3` | `Winding1`,`Winding2`,`Winding3` | `WindingType` | - | yes | Winding type at side 1/2/3. |
| `clock_12`,`clock_13` | `Clock12`,`Clock13` | `int8` | - | yes | Clock number of phase shift across side 1-2/1-3, `-12..12`. |
| `tap_side` | `TapSide` | `Branch3Side` | - | yes | Side of the tap changer (`Side1`/`Side2`/`Side3`). |
| `tap_pos` | `TapPos` | `int8` | - | no, default `tap_nom` (or `0`) | Current tap position. |
| `tap_min`,`tap_max` | `TapMin`,`TapMax` | `int8` | - | yes | Tap positions at minimum/maximum voltage. |
| `tap_nom` | `TapNom` | `int8` | - | no, default `0` | Nominal tap position. |
| `tap_size` | `TapSize` | `float64` | volt (V) | yes | Voltage size of each tap. Must be > 0. |
| `uk_12_min`/`_max`, `uk_13_min`/`_max`, `uk_23_min`/`_max` | `Uk12Min`/`Max`, `Uk13Min`/`Max`, `Uk23Min`/`Max` | `float64` | - | no, default the non-tap value | Relative short circuit voltage at minimum/maximum tap, per side pair. |
| `pk_12_min`/`_max`, `pk_13_min`/`_max`, `pk_23_min`/`_max` | `Pk12Min`/`Max`, `Pk13Min`/`Max`, `Pk23Min`/`Max` | `float64` | watt (W) | no, default the non-tap value | Short circuit loss at minimum/maximum tap, per side pair. |
| `r_grounding_1`/`2`/`3`, `x_grounding_1`/`2`/`3` | `RGrounding1`/`2`/`3`, `XGrounding1`/`2`/`3` | `float64` | ohm (Ω) | no, default `0` | Grounding resistance/reactance at side 1/2/3, if relevant. |

### Source

An appliance representing the external network as a Thévenin equivalent (infinite voltage source behind an
internal impedance specified as short circuit power). Reference direction: generator. Type name: `source`. Go:
`pgmtypes.Source` (embeds `ApplianceBase`: `Node`, `Status` — required).

| JSON field      | Go field      | Type      | Unit             | Required                | Description                                              |
|------------------|----------------|-----------|------------------|----------------------------|------------------------------------------------------------|
| `u_ref`         | `URef`        | `float64` | -                | only for power flow        | Reference voltage in per-unit. Must be > 0.                |
| `u_ref_angle`   | `URefAngle`   | `float64` | radian           | no, default `0.0`          | Reference voltage angle.                                    |
| `sk`            | `Sk`          | `float64` | volt-ampere (VA) | no, default `1e10`         | Short circuit power. Must be > 0.                           |
| `rx_ratio`      | `RxRatio`     | `float64` | -                | no, default `0.1`          | R to X ratio. Must be >= 0.                                 |
| `z01_ratio`     | `Z01Ratio`    | `float64` | -                | no, default `1.0`          | Zero-sequence to positive-sequence impedance ratio. Must be > 0. |

### Load / Generator (`sym_load`, `sym_gen`, `asym_load`, `asym_gen`)

An appliance representing a load or generator; the ZIP model type is chosen per-component via `type`. All four
type names share identical fields — they differ only in reference direction (load vs. generator) and whether
`p_specified`/`q_specified` are a single value or per-phase. Go: `pgmtypes.SymLoad`/`SymGen` (alias for
`LoadGen[float64]`), `pgmtypes.AsymLoad`/`AsymGen` (alias for `LoadGen[[3]float64]`); all embed `ApplianceBase`.

| JSON field      | Go field      | Type                        | Unit                       | Required             | Description |
|------------------|----------------|------------------------------|-----------------------------|-------------------------|--------------|
| `type`          | `Type`        | `LoadGenType`                | -                           | yes                     | Response to voltage: `ConstPower`, `ConstImpedance`, or `ConstCurrent`. |
| `p_specified`   | `PSpecified`  | `float64` (sym) / `[3]float64` (asym) | watt (W)             | only for power flow     | Specified active power. Reference direction: load or generator per type name. |
| `q_specified`   | `QSpecified`  | `float64` (sym) / `[3]float64` (asym) | volt-ampere-reactive (var) | only for power flow | Specified reactive power. |

### Shunt

An appliance with a fixed admittance, behaving like a `ConstImpedance` load; can also be used to introduce a
ground reference in an otherwise floating grid. Reference direction: load. Type name: `shunt`. Go:
`pgmtypes.Shunt` (embeds `ApplianceBase`).

| JSON field | Go field | Type      | Unit        | Required                          | Description                             |
|------------|----------|-----------|-------------|--------------------------------------|-------------------------------------------|
| `g1`       | `G1`     | `float64` | siemens (S) | yes                                   | Positive-sequence shunt conductance.       |
| `b1`       | `B1`     | `float64` | siemens (S) | yes                                   | Positive-sequence shunt susceptance.       |
| `g0`       | `G0`     | `float64` | siemens (S) | only for asymmetric calculations    | Zero-sequence shunt conductance.           |
| `b0`       | `B0`     | `float64` | siemens (S) | only for asymmetric calculations    | Zero-sequence shunt susceptance.           |

### Voltage Sensor (`sym_voltage_sensor`, `asym_voltage_sensor`)

A sensor measuring a node's voltage for state estimation (`sym_voltage_sensor`: line-to-line, single value;
`asym_voltage_sensor`: line-to-ground, per phase). Go: `pgmtypes.SymVoltageSensor`/`AsymVoltageSensor` (aliases
for `VoltageSensor[float64]`/`VoltageSensor[[3]float64]`; embeds `SensorBase`: `MeasuredObject` — required).

| JSON field           | Go field         | Type                        | Unit     | Required                                              | Description |
|------------------------|--------------------|------------------------------|----------|----------------------------------------------------------|--------------|
| `u_sigma`             | `USigma`          | `float64`                    | volt (V) | only for state estimation                                 | Standard deviation of the measurement error (usually the absolute error range / 3). Must be > 0. |
| `u_measured`          | `UMeasured`       | `float64` (sym) / `[3]float64` (asym) | volt (V) | only for state estimation                       | Measured voltage magnitude. Must be > 0. |
| `u_angle_measured`    | `UAngleMeasured`  | `float64` (sym) / `[3]float64` (asym) | radian   | only when a `GlobalAngle` current sensor is present | Measured voltage angle (phasor measurement units only). |

### Power Sensor (`sym_power_sensor`, `asym_power_sensor`)

A sensor measuring active/reactive power flow of a terminal (an appliance-node terminal, or a branch from/to
terminal — not `link`), for state estimation. Go: `pgmtypes.SymPowerSensor`/`AsymPowerSensor` (aliases for
`PowerSensor[float64]`/`PowerSensor[[3]float64]`; embeds `SensorBase`).

| JSON field                | Go field                | Type                        | Unit                       | Required                                             | Description |
|------------------------------|----------------------------|------------------------------|-----------------------------|---------------------------------------------------------|--------------|
| `measured_terminal_type`    | `MeasuredTerminalType`     | `MeasuredTerminalType`        | -                           | yes                                                        | Whether an appliance or branch terminal is measured; must match `measured_object`. |
| `power_sigma`               | `PowerSigma`               | `float64`                    | volt-ampere (VA)           | required unless both `p_sigma`/`q_sigma` given, for SE     | Standard deviation of the apparent power measurement error. Must be > 0. |
| `p_measured`, `q_measured`  | `PMeasured`, `QMeasured`   | `float64` (sym) / `[3]float64` (asym) | watt (W) / var         | only for state estimation                                  | Measured active/reactive power. |
| `p_sigma`, `q_sigma`        | `PSigma`, `QSigma`         | `float64` (sym) / `[3]float64` (asym) | watt (W) / var         | provide both or neither (see note below)                   | Standard deviation of the active/reactive power measurement error. Must be > 0. |

Note: if both `p_sigma` and `q_sigma` are given, they're used and `power_sigma` is ignored; if neither is given,
`power_sigma` is used instead; providing only one of `p_sigma`/`q_sigma` is undefined behavior.

### Current Sensor (`sym_current_sensor`, `asym_current_sensor`)

A sensor measuring the magnitude and angle of current flow of a branch (not `link`) terminal, for state
estimation. Go: `pgmtypes.SymCurrentSensor`/`AsymCurrentSensor` (aliases for
`CurrentSensor[float64]`/`CurrentSensor[[3]float64]`; embeds `SensorBase`).

| JSON field | Go field | Type | Unit | Required | Description |
| ------------------------------- | ----------------------------- | ------------------------------ | ------------ | --------------------------------- | -------------- |
| `measured_terminal_type` | `MeasuredTerminalType` | `MeasuredTerminalType` | - | yes | Which side of the branch is measured; must match `measured_object`. |
| `angle_measurement_type` | `AngleMeasurementType` | `AngleMeasurementType` | - | yes | Whether `i_angle_measured` is a global or local angle. |
| `i_sigma` | `ISigma` | `float64` | ampere (A) | only for state estimation | Standard deviation of the current magnitude measurement error. Must be > 0. |
| `i_angle_sigma` | `IAngleSigma` | `float64` | radian | only for state estimation | Standard deviation of the current angle measurement error. Must be > 0. |
| `i_measured` | `IMeasured` | `float64` (sym) / `[3]float64` (asym) | ampere (A) | only for state estimation | Measured current magnitude. |
| `i_angle_measured` | `IAngleMeasured` | `float64` (sym) / `[3]float64` (asym) | radian | only for state estimation | Measured current phase angle. `i_measured=0` with `i_angle_measured=nπ/2` is invalid. |

### Fault

A short circuit location; can only be placed at a `node`. Type name: `fault`. Go: `pgmtypes.Fault` (embeds
`Base`).

| JSON field     | Go field       | Type         | Unit    | Required                          | Description |
|-----------------|-----------------|--------------|---------|--------------------------------------|--------------|
| `status`       | `Status`       | `int8`       | -       | yes                                    | Whether the fault is active: `0` or `1`. |
| `fault_type`   | `FaultType`    | `FaultType`  | -       | only for short circuit                 | Type of the fault. |
| `fault_phase`  | `FaultPhase`   | `FaultPhase` | -       | no, default depends on `fault_type`    | Faulty phase(s) — see [Enum reference](#enum-reference). |
| `fault_object` | `FaultObject`  | `int32`      | -       | yes                                     | ID of the `node` where the short circuit happens. |
| `r_f`, `x_f`   | `RF`, `XF`     | `float64`    | ohm (Ω) | no, default `0.0`                      | Short circuit resistance/reactance. |

### Transformer Tap Regulator

Regulates the tap position of a `transformer` or `three_winding_transformer` to keep the voltage on its control
side within a band. Type name: `transformer_tap_regulator`. Go: `pgmtypes.TransformerTapRegulator` (embeds
`RegulatorBase`: `RegulatedObject` — ID of the transformer being regulated, `Status` — both required).

| JSON field                    | Go field                  | Type      | Unit     | Required             | Description |
|---------------------------------|------------------------------|-----------|----------|--------------------------|--------------|
| `control_side`                 | `ControlSide`                | `int8`    | -        | only for power flow        | Controlled side: use `BranchSide` constants for a `transformer`, `Branch3Side` constants for a `three_winding_transformer`. Should be the side relatively further from a source. |
| `u_set`                        | `USet`                       | `float64` | volt (V) | only for power flow        | Voltage setpoint at the center of the band. Must be >= 0. |
| `u_band`                       | `UBand`                      | `float64` | volt (V) | only for power flow        | Width of the voltage band. Must be > 0. |
| `line_drop_compensation_r`/`x` | `LineDropCompensationR`/`X`  | `float64` | ohm (Ω)  | no, default `0.0`          | Compensation for voltage drop due to resistance/reactance during transport. Must be >= 0. |

### Voltage Regulator

Defines voltage control for a regulated load/generator: an active regulator makes its node a voltage-controlled
PV node in Newton-Raphson power flow. Type name: `voltage_regulator`. Go: `pgmtypes.VoltageRegulator` (embeds
`RegulatorBase`: `RegulatedObject` — ID of the `sym_gen`/`asym_gen`/`sym_load`/`asym_load` being regulated,
`Status`).

| JSON field | Go field | Type      | Unit                       | Required             | Description                                                          |
|------------|----------|-----------|-----------------------------|--------------------------|------------------------------------------------------------------------|
| `u_ref`   | `URef`   | `float64` | -                           | only for power flow        | Reference voltage in per-unit at the regulated object's node. Must be > 0. |
| `q_min`   | `QMin`   | `float64` | volt-ampere-reactive (var) | no                          | Minimum reactive power limit of the regulated object. Unset = no limit.  |
| `q_max`   | `QMax`   | `float64` | volt-ampere-reactive (var) | no                          | Maximum reactive power limit of the regulated object. Unset = no limit.  |

## Enum reference

Every enum below is a JSON `int8` value; the Go type/constants are in
`examples/go_client/internal/pgmtypes/enums.go`.

| Go type                     | JSON value → Go constant                                                                                                                  |
|------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------|
| `LoadGenType`                | `0`=`ConstPower`, `1`=`ConstImpedance`, `2`=`ConstCurrent`                                                                                    |
| `WindingType`                | `0`=`WindingWye`, `1`=`WindingWyeN`, `2`=`WindingDelta`, `3`=`WindingZigzag`, `4`=`WindingZigzagN`                                            |
| `BranchSide`                 | `0`=`FromSide`, `1`=`ToSide`                                                                                                                   |
| `Branch3Side`                | `0`=`Side1`, `1`=`Side2`, `2`=`Side3`                                                                                                          |
| `MeasuredTerminalType`       | `0`=`BranchFrom`, `1`=`BranchTo`, `2`=`SourceTerminal`, `3`=`ShuntTerminal`, `4`=`LoadTerminal`, `5`=`GeneratorTerminal`, `6`=`Branch3Terminal1`, `7`=`Branch3Terminal2`, `8`=`Branch3Terminal3` (`9`=`NodeTerminal` is deprecated upstream) |
| `AngleMeasurementType`       | `0`=`LocalAngle`, `1`=`GlobalAngle`                                                                                                            |
| `FaultType`                  | `0`=`ThreePhase`, `1`=`SinglePhaseToGround`, `2`=`TwoPhase`, `3`=`TwoPhaseToGround`                                                            |
| `FaultPhase`                 | `0`=`FaultPhaseABC`, `1`=`FaultPhaseA`, `2`=`FaultPhaseB`, `3`=`FaultPhaseC`, `4`=`FaultPhaseAB`, `5`=`FaultPhaseAC`, `6`=`FaultPhaseBC`, `-1`=`DefaultFaultPhase` |
| `ShortCircuitVoltageScaling` | `0`=`VoltageScalingMinimum`, `1`=`VoltageScalingMaximum` (this one is a `CalculateShortCircuit` request option, not an input attribute)       |
