// SPDX-FileCopyrightText: Contributors to the Power Grid Model project <powergridmodel@lfenergy.org>
//
// SPDX-License-Identifier: MPL-2.0

package pgmtypes

// TransformerTapRegulator regulates the tap position of a Transformer or ThreeWindingTransformer to keep the
// voltage on its control side within a band. Type name: "transformer_tap_regulator". RegulatedObject (from
// RegulatorBase) must be the ID of the Transformer/ThreeWindingTransformer being regulated. See
// docs/user_manual/components.md#transformer-tap-regulator.
type TransformerTapRegulator struct {
	RegulatorBase
	// ControlSide is the controlled side of the transformer; required only for power flow. Use the BranchSide
	// constants (FromSide/ToSide) if RegulatedObject is a Transformer, or the Branch3Side constants
	// (Side1/Side2/Side3) if it is a ThreeWindingTransformer — both are int8-backed, hence the plain int8 type
	// here. Should be the relatively further side from a source.
	ControlSide *int8 `json:"control_side,omitempty"`
	// USet is the voltage setpoint at the center of the band, in volt (V); required only for power flow. Must
	// be >= 0.
	USet *float64 `json:"u_set,omitempty"`
	// UBand is the width of the voltage band (= 2 * acceptable deviation), in volt (V); required only for
	// power flow. Must be > 0.
	UBand *float64 `json:"u_band,omitempty"`
	// LineDropCompensationR/LineDropCompensationX compensate for voltage drop due to resistance/reactance
	// during transport to a virtual point further in the grid, in ohm (Ω). Default 0.0 each. Must be >= 0.
	LineDropCompensationR *float64 `json:"line_drop_compensation_r,omitempty"`
	LineDropCompensationX *float64 `json:"line_drop_compensation_x,omitempty"`
}

// VoltageRegulator defines voltage control for a regulated SymGen/AsymGen/SymLoad/AsymLoad: in Newton-Raphson
// power flow, an active voltage regulator makes its node a voltage-controlled PV node. Supported only by the
// Newton-Raphson power flow method. Type name: "voltage_regulator". RegulatedObject (from RegulatorBase) must
// be the ID of the regulated SymGen/AsymGen/SymLoad/AsymLoad. See
// docs/user_manual/components.md#voltage-regulator.
type VoltageRegulator struct {
	RegulatorBase
	// URef is the reference voltage in per-unit at the regulated object's node; required only for power flow.
	// Must be > 0.
	URef *float64 `json:"u_ref,omitempty"`
	// QMin/QMax are the minimum/maximum reactive power limit of the regulated object, in
	// volt-ampere-reactive (var). Unset means no limit.
	QMin *float64 `json:"q_min,omitempty"`
	QMax *float64 `json:"q_max,omitempty"`
}
