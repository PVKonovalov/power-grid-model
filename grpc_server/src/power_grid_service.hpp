// SPDX-FileCopyrightText: Contributors to the Power Grid Model project <powergridmodel@lfenergy.org>
//
// SPDX-License-Identifier: MPL-2.0

#pragma once

#include "proto/power_grid.grpc.pb.h"

#include <grpcpp/grpcpp.h>

#include <memory>

namespace power_grid_model_grpc {

// Holds live power_grid_model_cpp::Model instances keyed by model_id, for the CreateModel/UpdateAndCalculate*/
// DestroyModel RPCs. Defined in power_grid_service.cpp — forward-declared here so this header (included by
// main.cpp) doesn't need to pull in power_grid_model_cpp.
class ModelRegistry;

// Implements the PowerGridService RPCs. The Calculate* RPCs deserialize the request's PGM input dataset JSON,
// build a throwaway model, run the requested calculation, and serialize the result back to JSON. The
// CreateModel/UpdateAndCalculate*/DestroyModel RPCs instead keep a model alive across calls (via
// ModelRegistry), for pushing incremental updates (e.g. a changed load or tap position) without rebuilding
// topology on every call — see the "Stateful model session API" comment in proto/power_grid.proto. Every
// calculation, stateless or stateful, logs its wall-clock duration and node/branch count to stdout. See
// src/power_grid_service.cpp for the per-RPC details.
class PowerGridServiceImpl final : public power_grid_model::v1::PowerGridService::Service {
  public:
    PowerGridServiceImpl();
    ~PowerGridServiceImpl() override;

    // Runs a power flow calculation (symmetric or asymmetric) and returns the sym_output/asym_output dataset.
    grpc::Status CalculatePowerFlow(grpc::ServerContext* context,
                                    power_grid_model::v1::CalculatePowerFlowRequest const* request,
                                    power_grid_model::v1::CalculatePowerFlowResponse* response) override;

    // Runs a state estimation calculation and returns the sym_output/asym_output dataset. The input dataset is
    // expected to include sensor components (e.g. sym_voltage_sensor, sym_power_sensor).
    grpc::Status CalculateStateEstimation(grpc::ServerContext* context,
                                          power_grid_model::v1::CalculateStateEstimationRequest const* request,
                                          power_grid_model::v1::CalculateStateEstimationResponse* response) override;

    // Runs an IEC 60909 short circuit calculation and returns the sc_output dataset (always per-phase a/b/c).
    // The input dataset is expected to include at least one fault component.
    grpc::Status CalculateShortCircuit(grpc::ServerContext* context,
                                       power_grid_model::v1::CalculateShortCircuitRequest const* request,
                                       power_grid_model::v1::CalculateShortCircuitResponse* response) override;

    // Builds a model from the request's input dataset JSON and keeps it alive in the registry, returning a
    // freshly generated model_id for use by the RPCs below. The model has no expiry; call DestroyModel when
    // done with it.
    grpc::Status CreateModel(grpc::ServerContext* context, power_grid_model::v1::CreateModelRequest const* request,
                             power_grid_model::v1::CreateModelResponse* response) override;

    // Removes and frees the model for request's model_id. Safe to call even while a call against that model_id
    // is in flight elsewhere (the model object is only actually destroyed once nothing references it anymore).
    grpc::Status DestroyModel(grpc::ServerContext* context, power_grid_model::v1::DestroyModelRequest const* request,
                              power_grid_model::v1::DestroyModelResponse* response) override;

    // Applies request's update_dataset_json (if non-empty) to the model for model_id, runs a power flow
    // calculation, and returns the resulting sym_output/asym_output dataset.
    grpc::Status
    UpdateAndCalculatePowerFlow(grpc::ServerContext* context,
                               power_grid_model::v1::UpdateAndCalculatePowerFlowRequest const* request,
                               power_grid_model::v1::CalculatePowerFlowResponse* response) override;

    // As UpdateAndCalculatePowerFlow, but runs a state estimation calculation.
    grpc::Status UpdateAndCalculateStateEstimation(
        grpc::ServerContext* context, power_grid_model::v1::UpdateAndCalculateStateEstimationRequest const* request,
        power_grid_model::v1::CalculateStateEstimationResponse* response) override;

    // As UpdateAndCalculatePowerFlow, but runs an IEC 60909 short circuit calculation.
    grpc::Status
    UpdateAndCalculateShortCircuit(grpc::ServerContext* context,
                                   power_grid_model::v1::UpdateAndCalculateShortCircuitRequest const* request,
                                   power_grid_model::v1::CalculateShortCircuitResponse* response) override;

  private:
    std::unique_ptr<ModelRegistry> model_registry_;
};

} // namespace power_grid_model_grpc
