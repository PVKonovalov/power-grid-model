// SPDX-FileCopyrightText: Contributors to the Power Grid Model project <powergridmodel@lfenergy.org>
//
// SPDX-License-Identifier: MPL-2.0

#include "power_grid_service.hpp"

#include <grpcpp/ext/proto_server_reflection_plugin.h>
#include <grpcpp/grpcpp.h>

#include <atomic>
#include <chrono>
#include <csignal>
#include <cstdlib>
#include <iostream>
#include <string>
#include <thread>

namespace {

// Set (from a signal handler) to request a graceful shutdown; only ever written/read as a lock-free atomic, so
// it is safe to touch from signal-handler context.
std::atomic<bool> g_shutdown_requested{false};

// Signal handler for SIGINT/SIGTERM: only sets an atomic flag, so it stays within what's safe to do from
// signal-handler context. The actual shutdown happens on the main thread in main(), which polls the flag.
extern "C" void request_shutdown(int /*signal*/) { g_shutdown_requested.store(true, std::memory_order_relaxed); }

// Reads GRPC_PORT (default 50051) and returns "0.0.0.0:<port>".
std::string server_address() {
    std::string port = "50051";
    if (char const* env_port = std::getenv("GRPC_PORT"); env_port != nullptr) {
        port = env_port;
    }
    return "0.0.0.0:" + port;
}

// Reads GRPC_SHUTDOWN_TIMEOUT_SECONDS (default 10): how long to let in-flight RPCs finish on SIGINT/SIGTERM
// before the server forcefully cancels them.
std::chrono::seconds shutdown_timeout() {
    long seconds = 10;
    if (char const* env_timeout = std::getenv("GRPC_SHUTDOWN_TIMEOUT_SECONDS"); env_timeout != nullptr) {
        seconds = std::strtol(env_timeout, nullptr, 10);
    }
    return std::chrono::seconds{seconds};
}

} // namespace

// Starts power_grid_grpc_server: registers PowerGridServiceImpl (plus health checking and reflection, so
// grpcurl works without a compiled client) and serves on GRPC_PORT (default 50051) until SIGINT/SIGTERM.
// On SIGINT/SIGTERM, stops accepting new RPCs and gives in-flight ones up to GRPC_SHUTDOWN_TIMEOUT_SECONDS
// (default 10s) to finish before forcefully cancelling them and exiting — so an orchestrator (systemd, Docker,
// Kubernetes) can restart or roll out a new version without dropping in-flight requests.
int main() {
    std::string const address = server_address();

    power_grid_model_grpc::PowerGridServiceImpl service;

    grpc::EnableDefaultHealthCheckService(true);
    grpc::reflection::InitProtoReflectionServerBuilderPlugin();

    grpc::ServerBuilder builder;
    builder.AddListeningPort(address, grpc::InsecureServerCredentials());
    builder.RegisterService(&service);

    std::shared_ptr<grpc::Server> const server{builder.BuildAndStart()};
    if (server == nullptr) {
        std::cerr << "Failed to start server on " << address
                  << " (port already in use, or an invalid address/credentials)." << std::endl;
        return 1;
    }
    std::cout << "power_grid_grpc_server listening on " << address << std::endl;

    std::signal(SIGINT, request_shutdown);
    std::signal(SIGTERM, request_shutdown);

    // Server::Wait() blocks until Shutdown() is called, so it runs on its own thread while the main thread polls
    // for the signal-set flag and drives the actual Shutdown() call from ordinary (non-signal-handler) context.
    std::thread wait_thread{[server] { server->Wait(); }};

    while (!g_shutdown_requested.load(std::memory_order_relaxed)) {
        std::this_thread::sleep_for(std::chrono::milliseconds(200));
    }

    std::cout << "shutdown requested, draining in-flight RPCs (up to " << shutdown_timeout().count() << "s)..."
              << std::endl;
    server->Shutdown(std::chrono::system_clock::now() + shutdown_timeout());
    wait_thread.join();

    return 0;
}
