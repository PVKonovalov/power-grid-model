// SPDX-FileCopyrightText: Contributors to the Power Grid Model project <powergridmodel@lfenergy.org>
//
// SPDX-License-Identifier: MPL-2.0

package pgmtypes

// LoadGenType is the response of a load/generator to voltage (see src/power_grid_model/_core/enum.py
// LoadGenType). Marshals as its underlying int8 value in JSON, matching PGM's own C++ enum.
type LoadGenType int8

const (
	ConstPower     LoadGenType = 0
	ConstImpedance LoadGenType = 1
	ConstCurrent   LoadGenType = 2
)

// WindingType is a transformer/three-winding-transformer winding connection type.
type WindingType int8

const (
	// WindingWye is a star connection without an explicit neutral reference (no neutral/ground connection).
	WindingWye WindingType = 0
	// WindingWyeN is a star connection with an explicit neutral reference (provides neutral/ground connection).
	WindingWyeN WindingType = 1
	// WindingDelta is a delta connection (no neutral/ground connection).
	WindingDelta WindingType = 2
	// WindingZigzag is a zigzag/interconnected connection (no neutral/ground connection).
	WindingZigzag WindingType = 3
	// WindingZigzagN is a zigzag/interconnected connection with neutral (provides neutral/ground connection).
	WindingZigzagN WindingType = 4
)

// BranchSide identifies a side of a two-sided branch (transformer's tap_side, transformer_tap_regulator's
// control_side when regulating a transformer).
type BranchSide int8

const (
	FromSide BranchSide = 0
	ToSide   BranchSide = 1
)

// Branch3Side identifies a side of a three-winding-transformer (its tap_side, or
// transformer_tap_regulator's control_side when regulating a three_winding_transformer).
type Branch3Side int8

const (
	Side1 Branch3Side = 0
	Side2 Branch3Side = 1
	Side3 Branch3Side = 2
)

// MeasuredTerminalType indicates which kind of terminal a power/current sensor measures.
type MeasuredTerminalType int8

const (
	BranchFrom        MeasuredTerminalType = 0
	BranchTo          MeasuredTerminalType = 1
	SourceTerminal    MeasuredTerminalType = 2
	ShuntTerminal     MeasuredTerminalType = 3
	LoadTerminal      MeasuredTerminalType = 4
	GeneratorTerminal MeasuredTerminalType = 5
	Branch3Terminal1  MeasuredTerminalType = 6
	Branch3Terminal2  MeasuredTerminalType = 7
	Branch3Terminal3  MeasuredTerminalType = 8
	// NodeTerminal (value 9, node injection) is deprecated upstream in favor of the more specific terminal
	// types above; kept only for completeness with PGM's enum.
	NodeTerminal MeasuredTerminalType = 9
)

// AngleMeasurementType is the reference frame of a current sensor's measured angle.
type AngleMeasurementType int8

const (
	// LocalAngle measurements are relative to the local voltage angle at the same terminal.
	LocalAngle AngleMeasurementType = 0
	// GlobalAngle measurements are relative to the same global reference angle as voltage phasor measurements;
	// requires at least one voltage sensor with UAngleMeasured set somewhere in the grid.
	GlobalAngle AngleMeasurementType = 1
)

// FaultType is the type of short circuit fault represented by a Fault component.
type FaultType int8

const (
	ThreePhase          FaultType = 0
	SinglePhaseToGround FaultType = 1
	TwoPhase            FaultType = 2
	TwoPhaseToGround    FaultType = 3
)

// FaultPhase is the faulty phase(s) for a given FaultType; DefaultFaultPhase picks the FaultType-dependent
// default listed in docs/user_manual/components.md (Fault types, fault phases and default values).
type FaultPhase int8

const (
	FaultPhaseABC     FaultPhase = 0
	FaultPhaseA       FaultPhase = 1
	FaultPhaseB       FaultPhase = 2
	FaultPhaseC       FaultPhase = 3
	FaultPhaseAB      FaultPhase = 4
	FaultPhaseAC      FaultPhase = 5
	FaultPhaseBC      FaultPhase = 6
	DefaultFaultPhase FaultPhase = -1
)

// ShortCircuitVoltageScaling selects the IEC 60909 voltage factor for a short circuit calculation. This mirrors
// power_grid.proto's ShortCircuitVoltageScaling enum (used on the request), not an input dataset attribute —
// included here for convenience since it's part of the same overall calculation configuration.
type ShortCircuitVoltageScaling int8

const (
	VoltageScalingMinimum ShortCircuitVoltageScaling = 0
	VoltageScalingMaximum ShortCircuitVoltageScaling = 1
)
