# SPDX-FileCopyrightText: Contributors to the Power Grid Model project <powergridmodel@lfenergy.org>
#
# SPDX-License-Identifier: MPL-2.0

cmake_minimum_required(VERSION 3.25)

project(power_grid_model_grpc_server)

set(CMAKE_CXX_STANDARD 23)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)

find_package(power_grid_model REQUIRED CONFIG)
find_package(gRPC CONFIG REQUIRED)

add_library(power_grid_proto OBJECT proto/power_grid.proto)
target_link_libraries(power_grid_proto PUBLIC protobuf::libprotobuf gRPC::grpc++)
target_include_directories(power_grid_proto PUBLIC "${CMAKE_CURRENT_BINARY_DIR}")

protobuf_generate(TARGET power_grid_proto LANGUAGE cpp)
protobuf_generate(
    TARGET power_grid_proto
    LANGUAGE grpc
    GENERATE_EXTENSIONS .grpc.pb.h .grpc.pb.cc
    PLUGIN "protoc-gen-grpc=$<TARGET_FILE:gRPC::grpc_cpp_plugin>"
)

add_executable(power_grid_grpc_server src/main.cpp src/power_grid_service.cpp)
target_compile_definitions(power_grid_grpc_server PRIVATE PGM_ENABLE_EXPERIMENTAL)
target_link_libraries(
    power_grid_grpc_server
    PRIVATE
        power_grid_proto
        power_grid_model::power_grid_model_c
        power_grid_model::power_grid_model_cpp
        gRPC::grpc++_reflection
)
set_target_properties(power_grid_grpc_server PROPERTIES INSTALL_RPATH_USE_LINK_PATH TRUE)

install(TARGETS power_grid_grpc_server)
