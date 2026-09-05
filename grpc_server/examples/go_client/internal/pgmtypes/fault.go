// SPDX-FileCopyrightText: Contributors to the Power Grid Model project <powergridmodel@lfenergy.org>
//
// SPDX-License-Identifier: MPL-2.0

package pgmtypes

// Fault defines a short circuit location in the grid; can only be placed at a Node. Type name: "fault". See
// docs/user_manual/components.md#fault.
type Fault struct {
	Base
	// Status is whether the fault is active: 0 or 1.
	Status int8 `json:"status"`
	// FaultType is the type of the fault; required only for short circuit calculations.
	FaultType *FaultType `json:"fault_type,omitempty"`
	// FaultPhase is the faulty phase(s); default depends on FaultType (see
	// docs/user_manual/components.md#fault-types-fault-phases-and-default-values) if omitted or
	// DefaultFaultPhase.
	FaultPhase *FaultPhase `json:"fault_phase,omitempty"`
	// FaultObject is the ID of the Node where the short circuit happens.
	FaultObject int32 `json:"fault_object"`
	// RF/XF are the short circuit resistance/reactance, in ohm (Ω). Default 0.0 each.
	RF *float64 `json:"r_f,omitempty"`
	XF *float64 `json:"x_f,omitempty"`
}
