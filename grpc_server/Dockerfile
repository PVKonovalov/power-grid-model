# SPDX-FileCopyrightText: Contributors to the Power Grid Model project <powergridmodel@lfenergy.org>
#
# SPDX-License-Identifier: MPL-2.0

# Build context is the repository root (see docker-compose.yml's `context: ..`), because this also needs to
# build the core power_grid_model_c / power_grid_model_cpp library from ../power_grid_model_c.

########################################
# Stage 1: build and install power_grid_model_c / power_grid_model_cpp (the core C/C++ library)
########################################
FROM ubuntu:24.04 AS core-builder

# gcc-14 specifically: the core is C++23 and needs newer concepts support than Ubuntu 24.04's default gcc-13
# (matches this repo's own CI, .github/workflows/build-test-release.yml, which installs gcc-14 the same way).
RUN apt-get update && apt-get install -y --no-install-recommends \
        gcc-14 g++-14 cmake ninja-build \
        libboost-dev libeigen3-dev nlohmann-json3-dev libmsgpack-cxx-dev \
    && rm -rf /var/lib/apt/lists/*
ENV CC=gcc-14
ENV CXX=g++-14

WORKDIR /src
COPY CMakeLists.txt VERSION VERSION.license LICENSE ./
COPY cmake ./cmake
COPY LICENSES ./LICENSES
COPY power_grid_model_c ./power_grid_model_c

# Plain Release build (no dev/test targets, no sanitizers): PGM_ENABLE_DEV_BUILD defaults to OFF, so this only
# builds+installs the power_grid_model_c shared library and the power_grid_model_cpp headers.
RUN cmake -S . -B build -G Ninja \
        -DCMAKE_BUILD_TYPE=Release \
        -DCMAKE_INSTALL_PREFIX=/opt/power-grid-model \
    && cmake --build build --parallel \
    && cmake --install build

########################################
# Stage 2: build power_grid_grpc_server against the core library installed in stage 1
########################################
FROM ubuntu:24.04 AS server-builder

# Same gcc-14 as core-builder (see its comment); power_grid_model_cpp.hpp is header-only and gets compiled
# directly into this binary too.
RUN apt-get update && apt-get install -y --no-install-recommends \
        gcc-14 g++-14 cmake ninja-build \
        libprotobuf-dev protobuf-compiler libgrpc++-dev libgrpc-dev protobuf-compiler-grpc \
    && rm -rf /var/lib/apt/lists/*
ENV CC=gcc-14
ENV CXX=g++-14

COPY --from=core-builder /opt/power-grid-model /opt/power-grid-model

WORKDIR /src/grpc_server
COPY grpc_server/CMakeLists.txt ./
COPY grpc_server/proto ./proto
COPY grpc_server/src ./src

RUN cmake -S . -B build -G Ninja \
        -DCMAKE_BUILD_TYPE=Release \
        -Dpower_grid_model_DIR=/opt/power-grid-model/lib/cmake/power_grid_model \
    && cmake --build build --parallel

########################################
# Stage 3: grpc_health_probe, so the runtime image gets a working HEALTHCHECK without carrying curl/ca-certificates
# in the final image just to have fetched it.
########################################
FROM ubuntu:24.04 AS health-probe-downloader

ARG GRPC_HEALTH_PROBE_VERSION=v0.4.56
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl \
    && rm -rf /var/lib/apt/lists/* \
    && curl -fsSL -o /usr/local/bin/grpc_health_probe \
        "https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/${GRPC_HEALTH_PROBE_VERSION}/grpc_health_probe-linux-$(dpkg --print-architecture)" \
    && chmod +x /usr/local/bin/grpc_health_probe

########################################
# Stage 4: minimal runtime image — just the two binaries plus their actual runtime shared libraries, no
# compilers or headers.
########################################
FROM ubuntu:24.04 AS runtime

RUN apt-get update && apt-get install -y --no-install-recommends \
        libgrpc++1.51t64 libgrpc29t64 libprotobuf32t64 libprotobuf-lite32t64 \
        libabsl20220623t64 libre2-10 libcares2 libssl3t64 zlib1g \
    && rm -rf /var/lib/apt/lists/*

COPY --from=core-builder /opt/power-grid-model/lib/libpower_grid_model_c.so* /usr/local/lib/
COPY --from=server-builder /src/grpc_server/build/power_grid_grpc_server /usr/local/bin/power_grid_grpc_server
COPY --from=health-probe-downloader /usr/local/bin/grpc_health_probe /usr/local/bin/grpc_health_probe
RUN ldconfig

RUN useradd --system --no-create-home --shell /usr/sbin/nologin pgm
USER pgm

EXPOSE 50051
ENV GRPC_PORT=50051
# Checks the grpc.health.v1.Health service that main.cpp enables via EnableDefaultHealthCheckService(true).
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD ["grpc_health_probe", "-addr=localhost:50051"]
ENTRYPOINT ["/usr/local/bin/power_grid_grpc_server"]
