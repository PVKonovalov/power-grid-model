// SPDX-FileCopyrightText: Contributors to the Power Grid Model project <powergridmodel@lfenergy.org>
//
// SPDX-License-Identifier: MPL-2.0

// Package samplenetworks holds PGM input dataset JSON for the small networks the go_client examples (and the
// soak test) run calculations against, so each example doesn't carry its own copy.
package samplenetworks

// Feeder is PGM's own input dataset JSON schema for a small 110/35 kV substation feeder (see
// power_grid_model_c_example/power_flow_example.c), suitable for CalculatePowerFlow:
//
//	source_5 --node_1==(20 km 110 kV line_6)==node_2--[transformer_7: 110/35 kV, 25 MVA]--node_3==(10 km 35 kV
//	line_8)==node_4----sym_load_9
const Feeder = `{
  "version": "1.0",
  "type": "input",
  "is_batch": false,
  "attributes": {},
  "data": {
    "node": [
      {"id": 1, "u_rated": 110e3},
      {"id": 2, "u_rated": 110e3},
      {"id": 3, "u_rated": 35e3},
      {"id": 4, "u_rated": 35e3}
    ],
    "line": [
      {"id": 6, "from_node": 1, "to_node": 2, "from_status": 1, "to_status": 1,
       "r1": 2.400, "x1": 7.880, "c1": 194e-9, "tan1": 0,
       "r0": 7.200, "x0": 23.600, "c0": 97e-9, "tan0": 0, "i_n": 500.0},
      {"id": 8, "from_node": 3, "to_node": 4, "from_status": 1, "to_status": 1,
       "r1": 2.530, "x1": 3.630, "c1": 100e-9, "tan1": 0,
       "r0": 7.200, "x0": 11.500, "c0": 55e-9, "tan0": 0, "i_n": 350.0}
    ],
    "transformer": [
      {"id": 7, "from_node": 2, "to_node": 3, "from_status": 1, "to_status": 1,
       "u1": 110e3, "u2": 35e3, "sn": 25e6, "uk": 0.105, "pk": 130e3, "i0": 0.006, "p0": 25e3,
       "winding_from": 1, "winding_to": 2, "clock": 11,
       "tap_side": 0, "tap_pos": 0, "tap_min": -8, "tap_max": 8, "tap_nom": 0, "tap_size": 1375.0}
    ],
    "source": [
      {"id": 5, "node": 1, "status": 1, "u_ref": 1.0, "sk": 3000e6, "rx_ratio": 0.1}
    ],
    "sym_load": [
      {"id": 9, "node": 4, "status": 1, "type": 0, "p_specified": 10e6, "q_specified": 3.287e6}
    ]
  }
}`

// FeederWithFault is Feeder with a bolted three-phase fault added at node_4 (see
// power_grid_model_c_example/short_circuit_example.c), suitable for CalculateShortCircuit.
const FeederWithFault = `{
  "version": "1.0",
  "type": "input",
  "is_batch": false,
  "attributes": {},
  "data": {
    "node": [
      {"id": 1, "u_rated": 110e3},
      {"id": 2, "u_rated": 110e3},
      {"id": 3, "u_rated": 35e3},
      {"id": 4, "u_rated": 35e3}
    ],
    "line": [
      {"id": 6, "from_node": 1, "to_node": 2, "from_status": 1, "to_status": 1,
       "r1": 2.400, "x1": 7.880, "c1": 194e-9, "tan1": 0,
       "r0": 7.200, "x0": 23.600, "c0": 97e-9, "tan0": 0, "i_n": 500.0},
      {"id": 8, "from_node": 3, "to_node": 4, "from_status": 1, "to_status": 1,
       "r1": 2.530, "x1": 3.630, "c1": 100e-9, "tan1": 0,
       "r0": 7.200, "x0": 11.500, "c0": 55e-9, "tan0": 0, "i_n": 350.0}
    ],
    "transformer": [
      {"id": 7, "from_node": 2, "to_node": 3, "from_status": 1, "to_status": 1,
       "u1": 110e3, "u2": 35e3, "sn": 25e6, "uk": 0.105, "pk": 130e3, "i0": 0.006, "p0": 25e3,
       "winding_from": 1, "winding_to": 2, "clock": 11,
       "tap_side": 0, "tap_pos": 0, "tap_min": -8, "tap_max": 8, "tap_nom": 0, "tap_size": 1375.0}
    ],
    "source": [
      {"id": 5, "node": 1, "status": 1, "u_ref": 1.0, "sk": 3000e6, "rx_ratio": 0.1}
    ],
    "sym_load": [
      {"id": 9, "node": 4, "status": 1, "type": 0, "p_specified": 10e6, "q_specified": 3.287e6}
    ],
    "fault": [
      {"id": 10, "status": 1, "fault_type": 0, "fault_phase": 0, "fault_object": 4, "r_f": 0.0, "x_f": 0.0}
    ]
  }
}`

// SingleNodeWithVoltageSensor is PGM's own input dataset JSON schema for a single node with a source and one
// voltage sensor measurement (see
// tests/data/state_estimation/single-node-source-sym-voltage-sensor/input.json), suitable for
// CalculateStateEstimation. With only one measurement and no other constraint, the weighted least squares
// estimate reproduces the sensor's measured voltage exactly (u = 12345.0 V, u_angle = 0.1 rad).
const SingleNodeWithVoltageSensor = `{
  "version": "1.0",
  "type": "input",
  "is_batch": false,
  "attributes": {},
  "data": {
    "node": [
      {"id": 1, "u_rated": 10000.0}
    ],
    "source": [
      {"id": 2, "node": 1, "status": 1, "u_ref": 1.0}
    ],
    "sym_voltage_sensor": [
      {"id": 3, "measured_object": 1, "u_sigma": 100.0, "u_measured": 12345.0, "u_angle_measured": 0.1}
    ]
  }
}`
