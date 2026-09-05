// SPDX-FileCopyrightText: Contributors to the Power Grid Model project <powergridmodel@lfenergy.org>
//
// SPDX-License-Identifier: MPL-2.0

package pgmtypes

// Line is a Branch with specified serial impedance and shunt admittance (a cable is also modeled as Line). Can
// only connect two nodes with the same rated voltage. Type name: "line". See
// docs/user_manual/components.md#line.
type Line struct {
	BranchBase
	// R1/X1 are the positive-sequence serial resistance/reactance, in ohm (Ω). Cannot both be 0.0.
	R1 float64 `json:"r1"`
	X1 float64 `json:"x1"`
	// C1 is the positive-sequence shunt capacitance, in farad (F).
	C1 float64 `json:"c1"`
	// Tan1 is the positive-sequence shunt loss factor (tan δ).
	Tan1 float64 `json:"tan1"`
	// R0/X0 are the zero-sequence serial resistance/reactance, in ohm (Ω); required only for asymmetric
	// calculations. Cannot both be 0.0 if provided.
	R0 *float64 `json:"r0,omitempty"`
	X0 *float64 `json:"x0,omitempty"`
	// C0 is the zero-sequence shunt capacitance, in farad (F); required only for asymmetric calculations.
	C0 *float64 `json:"c0,omitempty"`
	// Tan0 is the zero-sequence shunt loss factor (tan δ); required only for asymmetric calculations.
	Tan0 *float64 `json:"tan0,omitempty"`
	// IN is the rated current, in ampere (A). Must be > 0 if provided; if omitted, "loading" in the output is NaN.
	IN *float64 `json:"i_n,omitempty"`
}

// AsymLine is a Branch with resistance/reactance/capacitance specified per phase (a/b/c, optionally n for
// neutral) rather than as symmetric-component (positive/zero sequence) parameters. Can only connect two nodes
// with the same rated voltage. Type name: "asym_line". See docs/user_manual/components.md#asym-line for the
// exact required-field combinations (full r/x matrix with or without neutral; c-matrix vs c0+c1).
type AsymLine struct {
	BranchBase
	// RAA/RBA/RBB/RCA/RCB/RCC are the series resistance matrix entries, in ohm (Ω): RAA/RBB/RCC must be > 0,
	// RBA/RCA/RCB must be >= 0.
	RAA float64 `json:"r_aa"`
	RBA float64 `json:"r_ba"`
	RBB float64 `json:"r_bb"`
	RCA float64 `json:"r_ca"`
	RCB float64 `json:"r_cb"`
	RCC float64 `json:"r_cc"`
	// RNA/RNB/RNC/RNN are the neutral-phase series resistance matrix entries, in ohm (Ω); required together (all
	// or none) if modeling a 4th (neutral) phase.
	RNA *float64 `json:"r_na,omitempty"`
	RNB *float64 `json:"r_nb,omitempty"`
	RNC *float64 `json:"r_nc,omitempty"`
	RNN *float64 `json:"r_nn,omitempty"`
	// XAA/XBA/XBB/XCA/XCB/XCC are the series reactance matrix entries, in ohm (Ω); same shape/validity rules as
	// the R matrix above.
	XAA float64 `json:"x_aa"`
	XBA float64 `json:"x_ba"`
	XBB float64 `json:"x_bb"`
	XCA float64 `json:"x_ca"`
	XCB float64 `json:"x_cb"`
	XCC float64 `json:"x_cc"`
	// XNA/XNB/XNC/XNN are the neutral-phase series reactance matrix entries, in ohm (Ω); required together (all
	// or none) if modeling a 4th (neutral) phase.
	XNA *float64 `json:"x_na,omitempty"`
	XNB *float64 `json:"x_nb,omitempty"`
	XNC *float64 `json:"x_nc,omitempty"`
	XNN *float64 `json:"x_nn,omitempty"`
	// CAA/CBA/CBB/CCA/CCB/CCC are the shunt nodal capacitance matrix entries, in farad (F); provide either the
	// full matrix or C0+C1 below (both cannot be omitted).
	CAA *float64 `json:"c_aa,omitempty"`
	CBA *float64 `json:"c_ba,omitempty"`
	CBB *float64 `json:"c_bb,omitempty"`
	CCA *float64 `json:"c_ca,omitempty"`
	CCB *float64 `json:"c_cb,omitempty"`
	CCC *float64 `json:"c_cc,omitempty"`
	// C0/C1 are the zero-/positive-sequence shunt capacitance, in farad (F); an alternative to the full C
	// matrix above (used in preference to it if both are given).
	C0 *float64 `json:"c0,omitempty"`
	C1 *float64 `json:"c1,omitempty"`
	// IN is the rated current, in ampere (A). Must be > 0 if provided; if omitted, "loading" in the output is NaN.
	IN *float64 `json:"i_n,omitempty"`
}

// Link is a Branch with a very high fixed admittance (effectively zero impedance), typically representing a
// short internal busbar connection. No sensors can be coupled to a Link. Has no additional attributes beyond
// BranchBase. Type name: "link". See docs/user_manual/components.md#link.
type Link struct {
	BranchBase
}

// GenericBranch is a Branch parameterized directly by its electrical equivalent-circuit (PI model) parameters,
// rather than transformer-style ratings; can behave like a line or a transformer depending on the chosen
// parameters. Asymmetric calculation is not supported for GenericBranch. Type name: "generic_branch". See
// docs/user_manual/components.md#generic-branch.
type GenericBranch struct {
	BranchBase
	// R1/X1 are the positive-sequence resistance/reactance, in ohm, referenced to the "to" side.
	R1 float64 `json:"r1"`
	X1 float64 `json:"x1"`
	// G1/B1 are the positive-sequence conductance/susceptance, in siemens, referenced to the "to" side.
	G1 float64 `json:"g1"`
	B1 float64 `json:"b1"`
	// K is the off-nominal ratio (not the nominal voltage ratio — must be set explicitly). Default 1.0. Must be > 0.
	K *float64 `json:"k,omitempty"`
	// Theta is the angle shift, in radian. Default 0.0.
	Theta *float64 `json:"theta,omitempty"`
	// Sn is the rated power, in volt-ampere (VA), used only for loading output. Default 0.0. Must be >= 0.
	Sn *float64 `json:"sn,omitempty"`
}

// Transformer is a Branch connecting two nodes at possibly different voltage levels. Type name: "transformer".
// See docs/user_manual/components.md#transformer.
type Transformer struct {
	BranchBase
	// U1/U2 are the rated voltage at the from-/to-side, in volt (V). Must be > 0.
	U1 float64 `json:"u1"`
	U2 float64 `json:"u2"`
	// Sn is the rated power, in volt-ampere (VA). Must be > 0.
	Sn float64 `json:"sn"`
	// Uk is the relative short circuit voltage (0.1 means 10%). Must be >= Pk/Sn, > 0, and < 1.
	Uk float64 `json:"uk"`
	// Pk is the short circuit (copper) loss, in watt (W). Must be >= 0.
	Pk float64 `json:"pk"`
	// I0 is the relative no-load (magnetizing) current. Must be >= P0/Sn and < 1.
	I0 float64 `json:"i0"`
	// P0 is the no-load (iron/magnetizing) loss, in watt (W). Must be >= 0.
	P0 float64 `json:"p0"`
	// I0ZeroSequence is the zero-sequence relative no-load current. Default: same as I0.
	I0ZeroSequence *float64 `json:"i0_zero_sequence,omitempty"`
	// P0ZeroSequence is the zero-sequence no-load loss, in watt (W). Default:
	// P0 + Pk*(I0ZeroSequence^2 - I0^2).
	P0ZeroSequence *float64 `json:"p0_zero_sequence,omitempty"`
	// WindingFrom/WindingTo are the from-/to-side winding types.
	WindingFrom WindingType `json:"winding_from"`
	WindingTo   WindingType `json:"winding_to"`
	// Clock is the clock number of phase shift, -12..12. Even numbers are invalid if exactly one side is a
	// Y(N) winding; odd numbers are invalid if both or neither side is a Y(N) winding.
	Clock int8 `json:"clock"`
	// TapSide is the side the tap changer is on.
	TapSide BranchSide `json:"tap_side"`
	// TapPos is the current tap position. Default: TapNom, or 0 if TapNom is also unset. Must be between
	// TapMin and TapMax (inclusive, in either order).
	TapPos *int8 `json:"tap_pos,omitempty"`
	// TapMin/TapMax are the tap positions at minimum/maximum voltage (note: TapMin may be > TapMax).
	TapMin int8 `json:"tap_min"`
	TapMax int8 `json:"tap_max"`
	// TapNom is the nominal tap position. Default 0. Must be between TapMin and TapMax (inclusive, in either
	// order).
	TapNom *int8 `json:"tap_nom,omitempty"`
	// TapSize is the voltage size of each tap, in volt (V). Must be >= 0.
	TapSize float64 `json:"tap_size"`
	// UkMin/UkMax are the relative short circuit voltage at minimum/maximum tap. Default: same as Uk.
	UkMin *float64 `json:"uk_min,omitempty"`
	UkMax *float64 `json:"uk_max,omitempty"`
	// PkMin/PkMax are the short circuit loss at minimum/maximum tap, in watt (W). Default: same as Pk.
	PkMin *float64 `json:"pk_min,omitempty"`
	PkMax *float64 `json:"pk_max,omitempty"`
	// RGroundingFrom/XGroundingFrom are the grounding resistance/reactance at the from-side, in ohm (Ω), if
	// relevant. Default 0.
	RGroundingFrom *float64 `json:"r_grounding_from,omitempty"`
	XGroundingFrom *float64 `json:"x_grounding_from,omitempty"`
	// RGroundingTo/XGroundingTo are the grounding resistance/reactance at the to-side, in ohm (Ω), if relevant.
	// Default 0.
	RGroundingTo *float64 `json:"r_grounding_to,omitempty"`
	XGroundingTo *float64 `json:"x_grounding_to,omitempty"`
}
