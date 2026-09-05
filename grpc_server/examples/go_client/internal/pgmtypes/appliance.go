// SPDX-FileCopyrightText: Contributors to the Power Grid Model project <powergridmodel@lfenergy.org>
//
// SPDX-License-Identifier: MPL-2.0

package pgmtypes

// LoadGen is an Appliance representing a load or generator. It's generic over P (RealValue) because the four
// concrete PGM types share identical fields and differ only in whether PSpecified/QSpecified are a single
// symmetric value or a 3-phase array, and in reference direction (load vs generator) — see SymLoad, SymGen,
// AsymLoad, AsymGen below and docs/user_manual/components.md#generic-load-and-generator.
type LoadGen[P RealValue] struct {
	ApplianceBase
	// Type is how the load/generator responds to voltage (ConstPower, ConstImpedance, or ConstCurrent).
	Type LoadGenType `json:"type"`
	// PSpecified/QSpecified are the specified active/reactive power, in watt (W) / volt-ampere-reactive (var);
	// required only for power flow. Reference direction is "load" for SymLoad/AsymLoad, "generator" for
	// SymGen/AsymGen (see data-model.md#reference-direction).
	PSpecified *P `json:"p_specified,omitempty"`
	QSpecified *P `json:"q_specified,omitempty"`
}

// SymLoad is a LoadGen with a single-phase-equivalent specified power and "load" reference direction. Type
// name: "sym_load".
type SymLoad = LoadGen[float64]

// SymGen is a LoadGen with a single-phase-equivalent specified power and "generator" reference direction. Type
// name: "sym_gen".
type SymGen = LoadGen[float64]

// AsymLoad is a LoadGen with a per-phase (a/b/c) specified power and "load" reference direction. Type name:
// "asym_load".
type AsymLoad = LoadGen[[3]float64]

// AsymGen is a LoadGen with a per-phase (a/b/c) specified power and "generator" reference direction. Type name:
// "asym_gen".
type AsymGen = LoadGen[[3]float64]

// Shunt is an Appliance with a fixed admittance, behaving like a LoadGen with Type ConstImpedance; also usable
// to introduce a ground reference in an otherwise floating grid. Reference direction: load. Type name: "shunt".
// See docs/user_manual/components.md#shunt.
type Shunt struct {
	ApplianceBase
	// G1/B1 are the positive-sequence shunt conductance/susceptance, in siemens (S).
	G1 float64 `json:"g1"`
	B1 float64 `json:"b1"`
	// G0/B0 are the zero-sequence shunt conductance/susceptance, in siemens (S); required only for asymmetric
	// calculations.
	G0 *float64 `json:"g0,omitempty"`
	B0 *float64 `json:"b0,omitempty"`
}

// Source is an Appliance representing the external network via a Thévenin equivalent: an infinite voltage
// source behind an internal impedance specified as short circuit power. Reference direction: generator. Type
// name: "source". See docs/user_manual/components.md#source.
type Source struct {
	ApplianceBase
	// URef is the reference voltage, in per-unit; required only for power flow. Must be > 0.
	URef *float64 `json:"u_ref,omitempty"`
	// URefAngle is the reference voltage angle, in radian. Default 0.0.
	URefAngle *float64 `json:"u_ref_angle,omitempty"`
	// Sk is the short circuit power, in volt-ampere (VA). Default 1e10. Must be > 0.
	Sk *float64 `json:"sk,omitempty"`
	// RxRatio is the R to X ratio. Default 0.1. Must be >= 0.
	RxRatio *float64 `json:"rx_ratio,omitempty"`
	// Z01Ratio is the zero-sequence to positive-sequence impedance ratio. Default 1.0. Must be > 0.
	Z01Ratio *float64 `json:"z01_ratio,omitempty"`
}
