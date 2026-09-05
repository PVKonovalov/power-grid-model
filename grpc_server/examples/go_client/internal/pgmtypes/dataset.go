// SPDX-FileCopyrightText: Contributors to the Power Grid Model project <powergridmodel@lfenergy.org>
//
// SPDX-License-Identifier: MPL-2.0

// Package pgmtypes provides typed Go structs mirroring PGM's own input dataset JSON schema (see
// ../../../README.md and docs/user_manual/components.md in the repo root), so a caller can build a
// CalculatePowerFlow/CalculateStateEstimation/CalculateShortCircuit request by populating Go structs instead of
// hand-writing JSON. Marshal an InputDataset with encoding/json to get the input_dataset_json string.
//
// Every optional attribute (anything with a documented default, or that's only required for certain calculation
// types) is a pointer field so it can be omitted (json:"...,omitempty") and left to PGM's own default, exactly
// as omitting the key from hand-written JSON would. Required fields are plain values. See the Ptr helper below
// for populating pointer fields concisely.
package pgmtypes

// RealValue is either a symmetric (single-phase-equivalent) or asymmetric (per-phase a/b/c) attribute value.
// Concrete component types are generic over this to avoid duplicating fields between e.g. SymLoad/AsymLoad or
// SymVoltageSensor/AsymVoltageSensor, which differ only in this respect (see docs/user_manual/components.md).
type RealValue interface {
	float64 | [3]float64
}

// InputDataset is the top-level envelope PGM's (de)serializer expects/produces, matching the schema used by
// power_grid_model_c_example/serialization.c.
type InputDataset struct {
	Version    string         `json:"version"`
	Type       string         `json:"type"`
	IsBatch    bool           `json:"is_batch"`
	Attributes map[string]any `json:"attributes"`
	Data       InputData      `json:"data"`
}

// NewInputDataset wraps data in an InputDataset with the envelope fields PGM expects for a (non-batch) input
// dataset: version "1.0", type "input", is_batch false, no attribute indications.
func NewInputDataset(data InputData) InputDataset {
	return InputDataset{
		Version:    "1.0",
		Type:       "input",
		IsBatch:    false,
		Attributes: map[string]any{},
		Data:       data,
	}
}

// InputData holds one slice per PGM component type. Unpopulated (nil) slices are omitted from the JSON, i.e. that
// component type is simply absent from the grid, matching how power_flow_example.c and short_circuit_example.c
// only ever declare the components a given grid actually has.
type InputData struct {
	Node                    []Node                    `json:"node,omitempty"`
	Line                    []Line                    `json:"line,omitempty"`
	AsymLine                []AsymLine                `json:"asym_line,omitempty"`
	Link                    []Link                    `json:"link,omitempty"`
	GenericBranch           []GenericBranch           `json:"generic_branch,omitempty"`
	Transformer             []Transformer             `json:"transformer,omitempty"`
	ThreeWindingTransformer []ThreeWindingTransformer `json:"three_winding_transformer,omitempty"`
	TransformerTapRegulator []TransformerTapRegulator `json:"transformer_tap_regulator,omitempty"`
	VoltageRegulator        []VoltageRegulator        `json:"voltage_regulator,omitempty"`
	SymLoad                 []SymLoad                 `json:"sym_load,omitempty"`
	SymGen                  []SymGen                  `json:"sym_gen,omitempty"`
	AsymLoad                []AsymLoad                `json:"asym_load,omitempty"`
	AsymGen                 []AsymGen                 `json:"asym_gen,omitempty"`
	Shunt                   []Shunt                   `json:"shunt,omitempty"`
	Source                  []Source                  `json:"source,omitempty"`
	SymVoltageSensor        []SymVoltageSensor        `json:"sym_voltage_sensor,omitempty"`
	AsymVoltageSensor       []AsymVoltageSensor       `json:"asym_voltage_sensor,omitempty"`
	SymPowerSensor          []SymPowerSensor          `json:"sym_power_sensor,omitempty"`
	AsymPowerSensor         []AsymPowerSensor         `json:"asym_power_sensor,omitempty"`
	SymCurrentSensor        []SymCurrentSensor        `json:"sym_current_sensor,omitempty"`
	AsymCurrentSensor       []AsymCurrentSensor       `json:"asym_current_sensor,omitempty"`
	Fault                   []Fault                   `json:"fault,omitempty"`
}

// Ptr returns a pointer to v, for populating optional pointer attribute fields inline without a temporary
// variable, e.g. Line{Tan1: pgmtypes.Ptr(0.0)} or Transformer{WindingFrom: pgmtypes.Ptr(WindingWyeN)}.
func Ptr[T any](v T) *T { return &v }
