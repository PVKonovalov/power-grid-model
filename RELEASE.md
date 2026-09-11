2026-09-05: Bumped `grpc_server`'s C++ standard from C++20 to C++23 (`grpc_server/CMakeLists.txt`) to match the
core library, updated `CLAUDE.md` and `grpc_server/README.md` accordingly (link's base fields now described as
"edge base fields" instead of "branch base fields"), and verified natively that `grpc_server` builds cleanly
against a freshly built/installed core with `-std=c++2b` actually applied.

2026-09-11: Added `power_grid_model_c_example/state_estimation.c`, a new C API example that deserializes an input
dataset (including voltage/power sensors) from a JSON file, runs a symmetric state estimation calculation
(iterative linear method), and serializes the result back to JSON. Bundled the input as
`power_grid_model_c_example/state_estimation_input.json` (a tracked copy of the ad-hoc `test.json`, with two
extra voltage sensors added on nodes 11 and 14 so every energized island in that network has the voltage
reference state estimation requires). Wired the new executable into
`power_grid_model_c_example/CMakeLists.txt` alongside the existing examples, registered as ctest
`PGMExampleStateEstimation`, and verified it builds warning-free and passes under the `apple-clang-debug`
preset.
