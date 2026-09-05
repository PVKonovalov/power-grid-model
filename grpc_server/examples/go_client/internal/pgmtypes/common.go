// SPDX-FileCopyrightText: Contributors to the Power Grid Model project <powergridmodel@lfenergy.org>
//
// SPDX-License-Identifier: MPL-2.0

package pgmtypes

// Base holds the one attribute every PGM component has. Embedded (not nested) in every component struct, so it
// flattens into the same JSON object — matching PGM's own "base" component (docs/user_manual/components.md#base).
type Base struct {
	// ID of this component. Must be unique across all components in the same scenario, e.g. you cannot have a
	// node with id=5 and a line with id=5.
	ID int32 `json:"id"`
}

// BranchBase holds the attributes shared by every two-sided branch (Line, AsymLine, Link, Transformer,
// GenericBranch): docs/user_manual/components.md#branch.
type BranchBase struct {
	Base
	// FromNode is the ID of a Node. Must reference a valid node.
	FromNode int32 `json:"from_node"`
	// ToNode is the ID of a Node. Must reference a valid node.
	ToNode int32 `json:"to_node"`
	// FromStatus is the connection status at the from-side: 0 or 1.
	FromStatus int8 `json:"from_status"`
	// ToStatus is the connection status at the to-side: 0 or 1.
	ToStatus int8 `json:"to_status"`
}

// Branch3Base holds the attributes shared by three-sided branches (currently only ThreeWindingTransformer):
// docs/user_manual/components.md#branch3.
type Branch3Base struct {
	Base
	// Node1/Node2/Node3 are the IDs of the Nodes at side 1/2/3. Must reference valid nodes.
	Node1 int32 `json:"node_1"`
	Node2 int32 `json:"node_2"`
	Node3 int32 `json:"node_3"`
	// Status1/Status2/Status3 are the connection statuses at side 1/2/3: 0 or 1.
	Status1 int8 `json:"status_1"`
	Status2 int8 `json:"status_2"`
	Status3 int8 `json:"status_3"`
}

// ApplianceBase holds the attributes shared by every appliance (LoadGen, Shunt, Source):
// docs/user_manual/components.md#appliance.
type ApplianceBase struct {
	Base
	// Node is the ID of the coupled Node. Must reference a valid node.
	Node int32 `json:"node"`
	// Status is the connection status to the node: 0 or 1.
	Status int8 `json:"status"`
}

// SensorBase holds the attribute shared by every sensor (VoltageSensor, PowerSensor, CurrentSensor):
// docs/user_manual/components.md#sensor.
type SensorBase struct {
	Base
	// MeasuredObject is the ID of the measured component. Must reference a valid object.
	MeasuredObject int32 `json:"measured_object"`
}

// RegulatorBase holds the attributes shared by every regulator (TransformerTapRegulator, VoltageRegulator):
// docs/user_manual/components.md#regulator.
type RegulatorBase struct {
	Base
	// RegulatedObject is the ID of the regulated component. Must reference a valid regulated object (a
	// Transformer/ThreeWindingTransformer for TransformerTapRegulator; a SymGen/AsymGen/SymLoad/AsymLoad for
	// VoltageRegulator).
	RegulatedObject int32 `json:"regulated_object"`
	// Status is the connection status to the regulated object: 0 or 1.
	Status int8 `json:"status"`
}
