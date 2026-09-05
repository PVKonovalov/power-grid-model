2026-09-05: Bumped `grpc_server`'s C++ standard from C++20 to C++23 (`grpc_server/CMakeLists.txt`) to match the
core library, updated `CLAUDE.md` and `grpc_server/README.md` accordingly (link's base fields now described as
"edge base fields" instead of "branch base fields"), and verified natively that `grpc_server` builds cleanly
against a freshly built/installed core with `-std=c++2b` actually applied.
