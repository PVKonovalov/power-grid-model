// SPDX-FileCopyrightText: Contributors to the Power Grid Model project <powergridmodel@lfenergy.org>
//
// SPDX-License-Identifier: MPL-2.0

#include "power_grid_service.hpp"

#include <power_grid_model_cpp.hpp>

#include <atomic>
#include <chrono>
#include <cstdint>
#include <iostream>
#include <memory>
#include <mutex>
#include <set>
#include <string>
#include <unordered_map>
#include <utility>

namespace power_grid_model_grpc {

namespace {

// Maps the proto CalculationMethod to the PGM_CalculationMethod values valid for power flow. Falls back to
// Newton-Raphson for CALCULATION_METHOD_UNSPECIFIED and for methods that aren't valid for power flow.
power_grid_model_cpp::Idx map_power_flow_method(power_grid_model::v1::CalculationMethod method) {
    switch (method) {
    case power_grid_model::v1::CALCULATION_METHOD_LINEAR:
        return PGM_linear;
    case power_grid_model::v1::CALCULATION_METHOD_ITERATIVE_CURRENT:
        return PGM_iterative_current;
    case power_grid_model::v1::CALCULATION_METHOD_NEWTON_RAPHSON:
    case power_grid_model::v1::CALCULATION_METHOD_UNSPECIFIED:
    default:
        return PGM_newton_raphson;
    }
}

// Maps the proto CalculationMethod to the PGM_CalculationMethod values valid for state estimation. Falls back
// to iterative linear for CALCULATION_METHOD_UNSPECIFIED and for methods that aren't valid for state estimation.
power_grid_model_cpp::Idx map_state_estimation_method(power_grid_model::v1::CalculationMethod method) {
    switch (method) {
    case power_grid_model::v1::CALCULATION_METHOD_NEWTON_RAPHSON:
        return PGM_newton_raphson;
    case power_grid_model::v1::CALCULATION_METHOD_ITERATIVE_LINEAR:
    case power_grid_model::v1::CALCULATION_METHOD_UNSPECIFIED:
    default:
        return PGM_iterative_linear;
    }
}

// Maps the proto ShortCircuitVoltageScaling to its PGM_ShortCircuitVoltageScaling counterpart, defaulting to
// maximum (the IEC 60909 c_max factor) for SHORT_CIRCUIT_VOLTAGE_SCALING_UNSPECIFIED.
power_grid_model_cpp::Idx
map_short_circuit_voltage_scaling(power_grid_model::v1::ShortCircuitVoltageScaling scaling) {
    switch (scaling) {
    case power_grid_model::v1::SHORT_CIRCUIT_VOLTAGE_SCALING_MINIMUM:
        return PGM_short_circuit_voltage_scaling_minimum;
    case power_grid_model::v1::SHORT_CIRCUIT_VOLTAGE_SCALING_MAXIMUM:
    case power_grid_model::v1::SHORT_CIRCUIT_VOLTAGE_SCALING_UNSPECIFIED:
    default:
        return PGM_short_circuit_voltage_scaling_maximum;
    }
}

// Counts "node" elements and elements of every two-/three-terminal branch component type in dataset_info, for
// the calculation-size log line below. Every input component type PGM supports is one of: node, a branch
// (line/asym_line/link/generic_branch/transformer), a branch3 (three_winding_transformer), an appliance
// (load/gen/shunt/source), a sensor, a fault, or a regulator — see docs/user_manual/components.md.
std::pair<power_grid_model_cpp::Idx, power_grid_model_cpp::Idx>
count_nodes_and_branches(power_grid_model_cpp::DatasetInfo const& dataset_info) {
    static std::set<std::string, std::less<>> const branch_component_names{
        "line", "asym_line", "link", "generic_branch", "transformer", "three_winding_transformer"};

    power_grid_model_cpp::Idx node_count{};
    power_grid_model_cpp::Idx branch_count{};
    for (power_grid_model_cpp::Idx component_idx{}; component_idx < dataset_info.n_components(); ++component_idx) {
        std::string const component_name = dataset_info.component_name(component_idx);
        power_grid_model_cpp::Idx const element_count = dataset_info.component_total_elements(component_idx);
        if (component_name == "node") {
            node_count += element_count;
        } else if (branch_component_names.count(component_name) > 0) {
            branch_count += element_count;
        }
    }
    return {node_count, branch_count};
}

// Logs one line per calculation: which RPC ran, the grid size it ran against, and how long the actual
// Model::calculate() call took (excluding JSON (de)serialization).
void log_calculation(char const* rpc_name, power_grid_model_cpp::Idx node_count, power_grid_model_cpp::Idx branch_count,
                     std::chrono::steady_clock::duration elapsed) {
    double const elapsed_ms = std::chrono::duration<double, std::milli>(elapsed).count();
    std::cout << "[calc] " << rpc_name << ": nodes=" << node_count << " branches=" << branch_count
              << " duration_ms=" << elapsed_ms << std::endl;
}

// Logs a successful CreateModel call: the model_id it was assigned, the grid size, and how long parsing the
// input dataset and constructing the Model (topology/Y-bus build) took. Uses a "[model]" prefix (rather than
// "[calc]") so model lifecycle events can be grepped for separately from calculation events.
void log_model_created(std::string const& model_id, power_grid_model_cpp::Idx node_count,
                       power_grid_model_cpp::Idx branch_count, std::chrono::steady_clock::duration elapsed) {
    double const elapsed_ms = std::chrono::duration<double, std::milli>(elapsed).count();
    std::cout << "[model] CreateModel: model_id=" << model_id << " nodes=" << node_count
              << " branches=" << branch_count << " duration_ms=" << elapsed_ms << std::endl;
}

// Logs a DestroyModel call and whether model_id actually existed.
void log_model_destroyed(std::string const& model_id, bool found) {
    std::cout << "[model] DestroyModel: model_id=" << model_id << " found=" << (found ? "true" : "false")
              << std::endl;
}

// One model kept alive across CreateModel/UpdateAndCalculate*/DestroyModel calls. `input` is retained (not just
// its values, but the OwningDataset itself) purely so its DatasetInfo can be reused to build correctly-shaped
// output datasets on every subsequent calculate call, without needing the original input JSON again.
// node_count/branch_count are computed once at CreateModel time, since the update datasets pushed later are
// partial and don't necessarily describe the whole grid.
class ManagedModel {
  public:
    ManagedModel(power_grid_model_cpp::OwningDataset&& input, power_grid_model_cpp::Model&& model,
                power_grid_model_cpp::Idx node_count, power_grid_model_cpp::Idx branch_count)
        : input_{std::move(input)}, model_{std::move(model)}, node_count_{node_count}, branch_count_{branch_count} {}

    // Guards every field below: calls against the same model_id are serialized through this mutex so that a
    // Model is never updated/calculated from two threads at once.
    std::mutex& mutex() { return mutex_; }
    power_grid_model_cpp::OwningDataset& input() { return input_; }
    power_grid_model_cpp::Model& model() { return model_; }
    power_grid_model_cpp::Idx node_count() const { return node_count_; }
    power_grid_model_cpp::Idx branch_count() const { return branch_count_; }

  private:
    std::mutex mutex_;
    power_grid_model_cpp::OwningDataset input_;
    power_grid_model_cpp::Model model_;
    power_grid_model_cpp::Idx node_count_;
    power_grid_model_cpp::Idx branch_count_;
};

} // namespace

// Maps model_id (a simple incrementing counter, not authenticated, not persisted across restarts) to a
// ManagedModel. Only ever hands out shared_ptr<ManagedModel>, so DestroyModel can safely remove an entry while
// another call still holds a reference to it — the model is freed once that call finishes and releases it.
class ModelRegistry {
  public:
    std::string create(power_grid_model_cpp::OwningDataset&& input, power_grid_model_cpp::Model&& model,
                       power_grid_model_cpp::Idx node_count, power_grid_model_cpp::Idx branch_count) {
        auto managed = std::make_shared<ManagedModel>(std::move(input), std::move(model), node_count, branch_count);
        std::string const model_id = std::to_string(next_id_.fetch_add(1, std::memory_order_relaxed));

        std::lock_guard<std::mutex> const lock{registry_mutex_};
        models_.emplace(model_id, std::move(managed));
        return model_id;
    }

    std::shared_ptr<ManagedModel> get(std::string const& model_id) {
        std::lock_guard<std::mutex> const lock{registry_mutex_};
        auto const it = models_.find(model_id);
        return it == models_.end() ? nullptr : it->second;
    }

    bool destroy(std::string const& model_id) {
        std::lock_guard<std::mutex> const lock{registry_mutex_};
        return models_.erase(model_id) > 0;
    }

  private:
    std::mutex registry_mutex_;
    std::unordered_map<std::string, std::shared_ptr<ManagedModel>> models_;
    std::atomic<std::uint64_t> next_id_{1};
};

PowerGridServiceImpl::PowerGridServiceImpl() : model_registry_{std::make_unique<ModelRegistry>()} {}

// Defined here (rather than defaulted in the header) because ~unique_ptr<ModelRegistry>() needs ModelRegistry to
// be a complete type, which it only is in this translation unit.
PowerGridServiceImpl::~PowerGridServiceImpl() = default;

// Deserializes request's input dataset JSON, builds a Model from it, runs a power flow calculation with the
// requested options, and serializes the resulting sym_output/asym_output dataset back into the response.
grpc::Status PowerGridServiceImpl::CalculatePowerFlow(grpc::ServerContext* /*context*/,
                                                      power_grid_model::v1::CalculatePowerFlowRequest const* request,
                                                      power_grid_model::v1::CalculatePowerFlowResponse* response) {
    using namespace power_grid_model_cpp;

    try {
        Deserializer deserializer{request->input_dataset_json(), PGM_json};
        OwningDataset input{deserializer.get_dataset()};
        deserializer.parse_to_buffer();

        auto const [node_count, branch_count] = count_nodes_and_branches(input.dataset.get_info());

        DatasetConst const input_const{input.dataset};
        Model model{request->system_frequency(), input_const};

        bool const sym = request->options().symmetric();
        OwningDataset output{input, PGM_power_flow, sym};

        Options options;
        options.set_calculation_type(PGM_power_flow);
        options.set_symmetric(sym ? PGM_symmetric : PGM_asymmetric);
        options.set_calculation_method(map_power_flow_method(request->options().calculation_method()));
        if (request->options().err_tol() > 0.0) {
            options.set_err_tol(request->options().err_tol());
        }
        if (request->options().max_iter() > 0) {
            options.set_max_iter(request->options().max_iter());
        }

        auto const start = std::chrono::steady_clock::now();
        model.calculate(options, output.dataset);
        log_calculation("CalculatePowerFlow", node_count, branch_count, std::chrono::steady_clock::now() - start);

        DatasetConst const output_const{output.dataset};
        Serializer serializer{output_const, PGM_json};
        response->set_output_dataset_json(serializer.get_to_zero_terminated_string(0, 2));
        return grpc::Status::OK;
    } catch (PowerGridError const& e) {
        return grpc::Status{grpc::StatusCode::INVALID_ARGUMENT, e.what()};
    } catch (std::exception const& e) {
        return grpc::Status{grpc::StatusCode::INTERNAL, e.what()};
    }
}

// Deserializes request's input dataset JSON (expected to include sensor components), builds a Model from it,
// runs a state estimation calculation with the requested options, and serializes the resulting
// sym_output/asym_output dataset back into the response.
grpc::Status PowerGridServiceImpl::CalculateStateEstimation(
    grpc::ServerContext* /*context*/, power_grid_model::v1::CalculateStateEstimationRequest const* request,
    power_grid_model::v1::CalculateStateEstimationResponse* response) {
    using namespace power_grid_model_cpp;

    try {
        Deserializer deserializer{request->input_dataset_json(), PGM_json};
        OwningDataset input{deserializer.get_dataset()};
        deserializer.parse_to_buffer();

        auto const [node_count, branch_count] = count_nodes_and_branches(input.dataset.get_info());

        DatasetConst const input_const{input.dataset};
        Model model{request->system_frequency(), input_const};

        bool const sym = request->options().symmetric();
        OwningDataset output{input, PGM_state_estimation, sym};

        Options options;
        options.set_calculation_type(PGM_state_estimation);
        options.set_symmetric(sym ? PGM_symmetric : PGM_asymmetric);
        options.set_calculation_method(map_state_estimation_method(request->options().calculation_method()));
        if (request->options().err_tol() > 0.0) {
            options.set_err_tol(request->options().err_tol());
        }
        if (request->options().max_iter() > 0) {
            options.set_max_iter(request->options().max_iter());
        }

        auto const start = std::chrono::steady_clock::now();
        model.calculate(options, output.dataset);
        log_calculation("CalculateStateEstimation", node_count, branch_count,
                        std::chrono::steady_clock::now() - start);

        DatasetConst const output_const{output.dataset};
        Serializer serializer{output_const, PGM_json};
        response->set_output_dataset_json(serializer.get_to_zero_terminated_string(0, 2));
        return grpc::Status::OK;
    } catch (PowerGridError const& e) {
        return grpc::Status{grpc::StatusCode::INVALID_ARGUMENT, e.what()};
    } catch (std::exception const& e) {
        return grpc::Status{grpc::StatusCode::INTERNAL, e.what()};
    }
}

// Deserializes request's input dataset JSON (expected to include at least one fault component), builds a
// Model from it, runs an IEC 60909 short circuit calculation with the requested voltage scaling, and
// serializes the resulting sc_output dataset back into the response.
grpc::Status
PowerGridServiceImpl::CalculateShortCircuit(grpc::ServerContext* /*context*/,
                                            power_grid_model::v1::CalculateShortCircuitRequest const* request,
                                            power_grid_model::v1::CalculateShortCircuitResponse* response) {
    using namespace power_grid_model_cpp;

    try {
        Deserializer deserializer{request->input_dataset_json(), PGM_json};
        OwningDataset input{deserializer.get_dataset()};
        deserializer.parse_to_buffer();

        auto const [node_count, branch_count] = count_nodes_and_branches(input.dataset.get_info());

        DatasetConst const input_const{input.dataset};
        Model model{request->system_frequency(), input_const};

        // The `sym` argument only affects the "sym_output"/"asym_output" choice; short circuit output is always
        // "sc_output" (see power_grid_model_cpp::get_output_type), so its value here is irrelevant.
        OwningDataset output{input, PGM_short_circuit, /*sym=*/true};

        Options options;
        options.set_calculation_type(PGM_short_circuit);
        options.set_calculation_method(PGM_iec60909); // the only calculation method PGM supports for short circuit
        options.set_short_circuit_voltage_scaling(
            map_short_circuit_voltage_scaling(request->options().voltage_scaling()));

        auto const start = std::chrono::steady_clock::now();
        model.calculate(options, output.dataset);
        log_calculation("CalculateShortCircuit", node_count, branch_count, std::chrono::steady_clock::now() - start);

        DatasetConst const output_const{output.dataset};
        Serializer serializer{output_const, PGM_json};
        response->set_output_dataset_json(serializer.get_to_zero_terminated_string(0, 2));
        return grpc::Status::OK;
    } catch (PowerGridError const& e) {
        return grpc::Status{grpc::StatusCode::INVALID_ARGUMENT, e.what()};
    } catch (std::exception const& e) {
        return grpc::Status{grpc::StatusCode::INTERNAL, e.what()};
    }
}

// Deserializes request's input dataset JSON, builds a Model from it, and stores the model (plus its node/branch
// counts, computed once here) in the registry under a freshly generated model_id.
grpc::Status PowerGridServiceImpl::CreateModel(grpc::ServerContext* /*context*/,
                                               power_grid_model::v1::CreateModelRequest const* request,
                                               power_grid_model::v1::CreateModelResponse* response) {
    using namespace power_grid_model_cpp;

    try {
        auto const start = std::chrono::steady_clock::now();

        Deserializer deserializer{request->input_dataset_json(), PGM_json};
        OwningDataset input{deserializer.get_dataset()};
        deserializer.parse_to_buffer();

        auto const [node_count, branch_count] = count_nodes_and_branches(input.dataset.get_info());

        DatasetConst const input_const{input.dataset};
        Model model{request->system_frequency(), input_const};

        std::string const model_id =
            model_registry_->create(std::move(input), std::move(model), node_count, branch_count);
        log_model_created(model_id, node_count, branch_count, std::chrono::steady_clock::now() - start);

        response->set_model_id(model_id);
        return grpc::Status::OK;
    } catch (PowerGridError const& e) {
        return grpc::Status{grpc::StatusCode::INVALID_ARGUMENT, e.what()};
    } catch (std::exception const& e) {
        return grpc::Status{grpc::StatusCode::INTERNAL, e.what()};
    }
}

// Removes request's model_id from the registry; NOT_FOUND if it doesn't exist.
grpc::Status PowerGridServiceImpl::DestroyModel(grpc::ServerContext* /*context*/,
                                                power_grid_model::v1::DestroyModelRequest const* request,
                                                power_grid_model::v1::DestroyModelResponse* /*response*/) {
    bool const found = model_registry_->destroy(request->model_id());
    log_model_destroyed(request->model_id(), found);
    if (!found) {
        return grpc::Status{grpc::StatusCode::NOT_FOUND, "no such model_id: " + request->model_id()};
    }
    return grpc::Status::OK;
}

// Looks up request's model_id (NOT_FOUND if missing), applies update_dataset_json to it if non-empty, runs a
// power flow calculation with the requested options, and serializes the resulting sym_output/asym_output
// dataset back into the response. Serialized against concurrent calls on the same model_id via its mutex.
grpc::Status PowerGridServiceImpl::UpdateAndCalculatePowerFlow(
    grpc::ServerContext* /*context*/, power_grid_model::v1::UpdateAndCalculatePowerFlowRequest const* request,
    power_grid_model::v1::CalculatePowerFlowResponse* response) {
    using namespace power_grid_model_cpp;

    auto const managed = model_registry_->get(request->model_id());
    if (managed == nullptr) {
        return grpc::Status{grpc::StatusCode::NOT_FOUND, "no such model_id: " + request->model_id()};
    }

    try {
        std::lock_guard<std::mutex> const lock{managed->mutex()};

        if (!request->update_dataset_json().empty()) {
            Deserializer deserializer{request->update_dataset_json(), PGM_json};
            OwningDataset update{deserializer.get_dataset()};
            deserializer.parse_to_buffer();
            DatasetConst const update_const{update.dataset};
            managed->model().update(update_const);
        }

        bool const sym = request->options().symmetric();
        OwningDataset output{managed->input(), PGM_power_flow, sym};

        Options options;
        options.set_calculation_type(PGM_power_flow);
        options.set_symmetric(sym ? PGM_symmetric : PGM_asymmetric);
        options.set_calculation_method(map_power_flow_method(request->options().calculation_method()));
        if (request->options().err_tol() > 0.0) {
            options.set_err_tol(request->options().err_tol());
        }
        if (request->options().max_iter() > 0) {
            options.set_max_iter(request->options().max_iter());
        }

        auto const start = std::chrono::steady_clock::now();
        managed->model().calculate(options, output.dataset);
        log_calculation("UpdateAndCalculatePowerFlow", managed->node_count(), managed->branch_count(),
                        std::chrono::steady_clock::now() - start);

        DatasetConst const output_const{output.dataset};
        Serializer serializer{output_const, PGM_json};
        response->set_output_dataset_json(serializer.get_to_zero_terminated_string(0, 2));
        return grpc::Status::OK;
    } catch (PowerGridError const& e) {
        return grpc::Status{grpc::StatusCode::INVALID_ARGUMENT, e.what()};
    } catch (std::exception const& e) {
        return grpc::Status{grpc::StatusCode::INTERNAL, e.what()};
    }
}

// As UpdateAndCalculatePowerFlow, but runs a state estimation calculation.
grpc::Status PowerGridServiceImpl::UpdateAndCalculateStateEstimation(
    grpc::ServerContext* /*context*/, power_grid_model::v1::UpdateAndCalculateStateEstimationRequest const* request,
    power_grid_model::v1::CalculateStateEstimationResponse* response) {
    using namespace power_grid_model_cpp;

    auto const managed = model_registry_->get(request->model_id());
    if (managed == nullptr) {
        return grpc::Status{grpc::StatusCode::NOT_FOUND, "no such model_id: " + request->model_id()};
    }

    try {
        std::lock_guard<std::mutex> const lock{managed->mutex()};

        if (!request->update_dataset_json().empty()) {
            Deserializer deserializer{request->update_dataset_json(), PGM_json};
            OwningDataset update{deserializer.get_dataset()};
            deserializer.parse_to_buffer();
            DatasetConst const update_const{update.dataset};
            managed->model().update(update_const);
        }

        bool const sym = request->options().symmetric();
        OwningDataset output{managed->input(), PGM_state_estimation, sym};

        Options options;
        options.set_calculation_type(PGM_state_estimation);
        options.set_symmetric(sym ? PGM_symmetric : PGM_asymmetric);
        options.set_calculation_method(map_state_estimation_method(request->options().calculation_method()));
        if (request->options().err_tol() > 0.0) {
            options.set_err_tol(request->options().err_tol());
        }
        if (request->options().max_iter() > 0) {
            options.set_max_iter(request->options().max_iter());
        }

        auto const start = std::chrono::steady_clock::now();
        managed->model().calculate(options, output.dataset);
        log_calculation("UpdateAndCalculateStateEstimation", managed->node_count(), managed->branch_count(),
                        std::chrono::steady_clock::now() - start);

        DatasetConst const output_const{output.dataset};
        Serializer serializer{output_const, PGM_json};
        response->set_output_dataset_json(serializer.get_to_zero_terminated_string(0, 2));
        return grpc::Status::OK;
    } catch (PowerGridError const& e) {
        return grpc::Status{grpc::StatusCode::INVALID_ARGUMENT, e.what()};
    } catch (std::exception const& e) {
        return grpc::Status{grpc::StatusCode::INTERNAL, e.what()};
    }
}

// As UpdateAndCalculatePowerFlow, but runs an IEC 60909 short circuit calculation.
grpc::Status PowerGridServiceImpl::UpdateAndCalculateShortCircuit(
    grpc::ServerContext* /*context*/, power_grid_model::v1::UpdateAndCalculateShortCircuitRequest const* request,
    power_grid_model::v1::CalculateShortCircuitResponse* response) {
    using namespace power_grid_model_cpp;

    auto const managed = model_registry_->get(request->model_id());
    if (managed == nullptr) {
        return grpc::Status{grpc::StatusCode::NOT_FOUND, "no such model_id: " + request->model_id()};
    }

    try {
        std::lock_guard<std::mutex> const lock{managed->mutex()};

        if (!request->update_dataset_json().empty()) {
            Deserializer deserializer{request->update_dataset_json(), PGM_json};
            OwningDataset update{deserializer.get_dataset()};
            deserializer.parse_to_buffer();
            DatasetConst const update_const{update.dataset};
            managed->model().update(update_const);
        }

        // See CalculateShortCircuit: the sym argument here is irrelevant, short circuit output is always sc_output.
        OwningDataset output{managed->input(), PGM_short_circuit, /*sym=*/true};

        Options options;
        options.set_calculation_type(PGM_short_circuit);
        options.set_calculation_method(PGM_iec60909);
        options.set_short_circuit_voltage_scaling(
            map_short_circuit_voltage_scaling(request->options().voltage_scaling()));

        auto const start = std::chrono::steady_clock::now();
        managed->model().calculate(options, output.dataset);
        log_calculation("UpdateAndCalculateShortCircuit", managed->node_count(), managed->branch_count(),
                        std::chrono::steady_clock::now() - start);

        DatasetConst const output_const{output.dataset};
        Serializer serializer{output_const, PGM_json};
        response->set_output_dataset_json(serializer.get_to_zero_terminated_string(0, 2));
        return grpc::Status::OK;
    } catch (PowerGridError const& e) {
        return grpc::Status{grpc::StatusCode::INVALID_ARGUMENT, e.what()};
    } catch (std::exception const& e) {
        return grpc::Status{grpc::StatusCode::INTERNAL, e.what()};
    }
}

} // namespace power_grid_model_grpc
