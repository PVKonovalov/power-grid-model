# SPDX-FileCopyrightText: Contributors to the Power Grid Model project <powergridmodel@lfenergy.org>
#
# SPDX-License-Identifier: MPL-2.0

add_executable(power_grid_model_c_example main.c)
target_link_libraries(power_grid_model_c_example PRIVATE power_grid_model_c)

add_executable(power_grid_model_c_example_serialization serialization.c)
target_link_libraries(power_grid_model_c_example_serialization PRIVATE power_grid_model_c)

add_executable(power_grid_model_c_example_state_estimation state_estimation.c)
target_link_libraries(power_grid_model_c_example_state_estimation PRIVATE power_grid_model_c)
target_compile_definitions(
    power_grid_model_c_example_state_estimation
    PRIVATE
        PGM_EXAMPLE_STATE_ESTIMATION_INPUT_JSON="${CMAKE_CURRENT_SOURCE_DIR}/state_estimation_input.json"
)

add_test(
    PGMExample
    ${CMAKE_RUNTIME_OUTPUT_DIRECTORY}/power_grid_model_c_example
)

add_test(
    PGMExampleSerialization
    ${CMAKE_RUNTIME_OUTPUT_DIRECTORY}/power_grid_model_c_example_serialization
)

add_test(
    PGMExampleStateEstimation
    ${CMAKE_RUNTIME_OUTPUT_DIRECTORY}/power_grid_model_c_example_state_estimation
)
