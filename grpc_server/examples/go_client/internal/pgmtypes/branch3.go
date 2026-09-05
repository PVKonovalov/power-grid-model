// SPDX-FileCopyrightText: Contributors to the Power Grid Model project <powergridmodel@lfenergy.org>
//
// SPDX-License-Identifier: MPL-2.0

package pgmtypes

// ThreeWindingTransformer is a Branch3 connecting three nodes at possibly different voltage levels. Type name:
// "three_winding_transformer". See docs/user_manual/components.md#three-winding-transformer.
type ThreeWindingTransformer struct {
	Branch3Base
	// U1/U2/U3 are the rated voltage at side 1/2/3, in volt (V). Must be > 0.
	U1 float64 `json:"u1"`
	U2 float64 `json:"u2"`
	U3 float64 `json:"u3"`
	// Sn1/Sn2/Sn3 are the rated power at side 1/2/3, in volt-ampere (VA). Must be > 0.
	Sn1 float64 `json:"sn_1"`
	Sn2 float64 `json:"sn_2"`
	Sn3 float64 `json:"sn_3"`
	// Uk12/Uk13/Uk23 are the relative short circuit voltage across side 1-2/1-3/2-3 (0.1 means 10%).
	Uk12 float64 `json:"uk_12"`
	Uk13 float64 `json:"uk_13"`
	Uk23 float64 `json:"uk_23"`
	// Pk12/Pk13/Pk23 are the short circuit (copper) loss across side 1-2/1-3/2-3, in watt (W). Must be >= 0.
	Pk12 float64 `json:"pk_12"`
	Pk13 float64 `json:"pk_13"`
	Pk23 float64 `json:"pk_23"`
	// I0 is the relative no-load (magnetizing) current with respect to side 1. Must be >= P0/Sn1 and < 1.
	I0 float64 `json:"i0"`
	// P0 is the no-load (iron/magnetizing) loss, in watt (W). Must be >= 0.
	P0 float64 `json:"p0"`
	// Winding1/Winding2/Winding3 are the side 1/2/3 winding types.
	Winding1 WindingType `json:"winding_1"`
	Winding2 WindingType `json:"winding_2"`
	Winding3 WindingType `json:"winding_3"`
	// Clock12/Clock13 are the clock numbers of phase shift across side 1-2/1-3, -12..12; an odd number is only
	// allowed for a Dy(n) or Y(N)d configuration.
	Clock12 int8 `json:"clock_12"`
	Clock13 int8 `json:"clock_13"`
	// TapSide is the side the tap changer is on: Side1, Side2, or Side3.
	TapSide Branch3Side `json:"tap_side"`
	// TapPos is the current tap position. Default: TapNom, or 0 if TapNom is also unset.
	TapPos *int8 `json:"tap_pos,omitempty"`
	// TapMin/TapMax are the tap positions at minimum/maximum voltage (note: TapMin may be > TapMax).
	TapMin int8 `json:"tap_min"`
	TapMax int8 `json:"tap_max"`
	// TapNom is the nominal tap position. Default 0.
	TapNom *int8 `json:"tap_nom,omitempty"`
	// TapSize is the voltage size of each tap, in volt (V). Must be > 0.
	TapSize float64 `json:"tap_size"`
	// Uk12Min/Uk12Max/Pk12Min/Pk12Max, Uk13Min/Uk13Max/Pk13Min/Pk13Max, Uk23Min/Uk23Max/Pk23Min/Pk23Max are the
	// tap-dependent short circuit voltage/loss across each side pair at minimum/maximum tap. Default: same as
	// the corresponding non-tap-dependent attribute above (Uk12/Pk12, etc.).
	Uk12Min *float64 `json:"uk_12_min,omitempty"`
	Uk12Max *float64 `json:"uk_12_max,omitempty"`
	Pk12Min *float64 `json:"pk_12_min,omitempty"`
	Pk12Max *float64 `json:"pk_12_max,omitempty"`
	Uk13Min *float64 `json:"uk_13_min,omitempty"`
	Uk13Max *float64 `json:"uk_13_max,omitempty"`
	Pk13Min *float64 `json:"pk_13_min,omitempty"`
	Pk13Max *float64 `json:"pk_13_max,omitempty"`
	Uk23Min *float64 `json:"uk_23_min,omitempty"`
	Uk23Max *float64 `json:"uk_23_max,omitempty"`
	Pk23Min *float64 `json:"pk_23_min,omitempty"`
	Pk23Max *float64 `json:"pk_23_max,omitempty"`
	// RGrounding1/XGrounding1, RGrounding2/XGrounding2, RGrounding3/XGrounding3 are the grounding
	// resistance/reactance at side 1/2/3, in ohm (Ω), if relevant. Default 0.
	RGrounding1 *float64 `json:"r_grounding_1,omitempty"`
	XGrounding1 *float64 `json:"x_grounding_1,omitempty"`
	RGrounding2 *float64 `json:"r_grounding_2,omitempty"`
	XGrounding2 *float64 `json:"x_grounding_2,omitempty"`
	RGrounding3 *float64 `json:"r_grounding_3,omitempty"`
	XGrounding3 *float64 `json:"x_grounding_3,omitempty"`
}
