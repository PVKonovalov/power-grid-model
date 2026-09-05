// SPDX-FileCopyrightText: Contributors to the Power Grid Model project <powergridmodel@lfenergy.org>
//
// SPDX-License-Identifier: MPL-2.0

package pgmtypes

// Node is a point in the grid (a busbar, a joint, or similar). Type name in the input dataset: "node".
type Node struct {
	Base
	// URated is the rated line-line voltage, in volt (V). Must be > 0.
	URated float64 `json:"u_rated"`
}
