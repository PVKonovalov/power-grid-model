# Building the C library and C examples on macOS

These are the exact steps used to build the `power_grid_model_c` shared library, the C++ unit/validation tests, and
the C API examples on this machine.

System used: macOS 26.5.2 (Tahoe), Apple Silicon (`arm64`), Apple Clang 21.0.0, CMake 3.30.3.

## 1. Install build dependencies

`boost`, `eigen`, and `cmake` were already installed via Homebrew. The following were missing and installed:

```shell
brew install ninja nlohmann-json msgpack-cxx doctest
```

Note: installing `msgpack-cxx` pulled in an upgrade of `boost` as a dependency.

## 2. Set environment variables

```shell
export CXX=clang++
export CC=clang
export CMAKE_PREFIX_PATH=/opt/homebrew   # so CMake finds the Homebrew-installed C++ packages
```

## 3. Pick a preset

```shell
cmake --list-presets
```

On macOS this gives `apple-clang-debug` / `apple-clang-release` (plus `clang-tidy-*` and `ci-*` variants). We used
`apple-clang-release`.

## 4. Build, test, and run the C API example

```shell
./build.sh -p apple-clang-release -e
```

This does, in order (see `build.sh`):

```shell
cmake --preset apple-clang-release
cmake --build --preset apple-clang-release --verbose -j1
ctest --preset apple-clang-release -E PGMExample --output-on-failure
ctest --preset apple-clang-release -R PGMExample --output-on-failure   # because -e was passed
```

Build output:

* Shared library: `cpp_build/apple-clang-release/bin/libpower_grid_model_c.dylib`
  (plus versioned symlinks `libpower_grid_model_c.1.dylib`, `libpower_grid_model_c.1.13.dylib`)
* This also builds `tests/cpp_unit_tests`, `tests/cpp_validation_tests`, `tests/native_api_tests`,
  `tests/benchmark_cpp`, and `power_grid_model_c_example` (developer build targets, see
  `docs/advanced_documentation/build-guide.md`).

### Result (first run) and the fix

194/195 tests passed on the first run. The one failure was:

```text
195 - Validation test batch - short circuit (Failed)
```

Specifically `short_circuit/dummy-test-line-into-itself-asym-iec60909_batch`, with mismatches like:

```text
Component: source #0 attribute: i: actual = (63508529.61085884, 731.3275936212143, 734.2600609542336)
                              vs. expected = (63508529.61085884, 731.3275922792482, 734.2600609406309)
```

`build.sh` uses `set -e`, so this `ctest` failure stopped the script before it reached the `-e` (C API example) step.

**Root cause:** this was a real bug in the test data, not a numeric/platform issue. The tolerance config in
`tests/data/short_circuit/dummy-test-line-into-itself/params.json` used the regex `"i_(.+)?"` to give the current
magnitude attribute a looser tolerance. That regex only matches names like `i_from`/`i_to` (it requires a literal
underscore right after `i`) — it does **not** match the bare `i` attribute, which is exactly the attribute that
mismatched. Every other short-circuit test case (e.g. `single_phase_to_ground_c_maximum/params.json`,
`two_phase_to_ground_c_minimum/params.json`) uses the pattern `"i(_.+)?"` instead, which matches `i` itself *or*
`i_xxx`. Because of the missing underscore-optionality, `i` silently fell back to the far stricter `"default": 1e-9`
tolerance instead of the intended `1e-5`, so a harmless ~`1.3e-6` numeric difference was flagged as a failure.

Fix applied in `tests/data/short_circuit/dummy-test-line-into-itself/params.json`:

```diff
-    "i_(.+)?": 1e-5
+    "i(_.+)?": 1e-5
```

The C API example step was then run manually (with the same environment variables set) and passed cleanly, and a
subsequent full `ctest` run confirmed all 197 tests pass:

```shell
ctest --preset apple-clang-release --output-on-failure
```

```text
100% tests passed, 0 tests failed out of 197
```

## Re-running only the example after a build

Once the project has been configured/built once, you don't need to rebuild to re-run the example:

```shell
ctest --preset apple-clang-release -R PGMExample --output-on-failure
```

## Clean full re-run

```shell
./build.sh -p apple-clang-release -e
```

This should now complete end-to-end (build + all 195 unit/validation tests + both C API examples) without stopping
early.
