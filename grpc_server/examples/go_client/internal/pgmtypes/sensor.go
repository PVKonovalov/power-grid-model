// SPDX-FileCopyrightText: Contributors to the Power Grid Model project <powergridmodel@lfenergy.org>
//
// SPDX-License-Identifier: MPL-2.0

package pgmtypes

// VoltageSensor is a Sensor measuring the magnitude and (optionally) angle of a Node's voltage, for use by
// state estimation. Generic over P (RealValue): SymVoltageSensor measures a line-to-line voltage as a single
// value; AsymVoltageSensor measures a 3-phase line-to-ground voltage per phase. See
// docs/user_manual/components.md#generic-voltage-sensor.
type VoltageSensor[P RealValue] struct {
	SensorBase
	// USigma is the standard deviation of the measurement error, in volt (V) — usually the absolute measurement
	// error range divided by 3. Required only for state estimation. Must be > 0.
	USigma *float64 `json:"u_sigma,omitempty"`
	// UMeasured is the measured voltage magnitude, in volt (V). Required only for state estimation. Must be > 0.
	UMeasured *P `json:"u_measured,omitempty"`
	// UAngleMeasured is the measured voltage angle, in radian (only meaningful with phasor measurement units).
	// Required only for state estimation when a current sensor with GlobalAngle AngleMeasurementType is present
	// elsewhere in the grid (it then serves as that measurement's reference angle).
	UAngleMeasured *P `json:"u_angle_measured,omitempty"`
}

// SymVoltageSensor is a VoltageSensor measuring a single (line-to-line) voltage value. Type name:
// "sym_voltage_sensor".
type SymVoltageSensor = VoltageSensor[float64]

// AsymVoltageSensor is a VoltageSensor measuring a 3-phase (line-to-ground) voltage, one value per phase a/b/c.
// Type name: "asym_voltage_sensor".
type AsymVoltageSensor = VoltageSensor[[3]float64]

// PowerSensor is a Sensor measuring the active/reactive power flow of a terminal (between an appliance and a
// node, or the from/to end of a branch other than Link, and a node), for use by state estimation. Generic over
// P (RealValue): SymPowerSensor measures a single value; AsymPowerSensor measures per-phase (a/b/c) values. See
// docs/user_manual/components.md#generic-power-sensor.
type PowerSensor[P RealValue] struct {
	SensorBase
	// MeasuredTerminalType indicates which kind of terminal is measured; must match what MeasuredObject
	// actually is.
	MeasuredTerminalType MeasuredTerminalType `json:"measured_terminal_type"`
	// PowerSigma is the standard deviation of the apparent power measurement error, in volt-ampere (VA) —
	// usually the absolute measurement error range divided by 3. Used when PSigma/QSigma are not both given
	// (see PSigma/QSigma below); required in that case for state estimation. Must be > 0.
	PowerSigma *float64 `json:"power_sigma,omitempty"`
	// PMeasured/QMeasured are the measured active/reactive power, in watt (W) / volt-ampere-reactive (var).
	// Required only for state estimation.
	PMeasured *P `json:"p_measured,omitempty"`
	QMeasured *P `json:"q_measured,omitempty"`
	// PSigma/QSigma are the standard deviation of the active/reactive power measurement error. Either provide
	// both (in which case PowerSigma above is ignored) or neither (in which case PowerSigma is required); see
	// docs/user_manual/components.md#generic-power-sensor for the full validity table. Must be > 0.
	PSigma *P `json:"p_sigma,omitempty"`
	QSigma *P `json:"q_sigma,omitempty"`
}

// SymPowerSensor is a PowerSensor measuring a single power value. Type name: "sym_power_sensor".
type SymPowerSensor = PowerSensor[float64]

// AsymPowerSensor is a PowerSensor measuring per-phase (a/b/c) power values. Type name: "asym_power_sensor".
type AsymPowerSensor = PowerSensor[[3]float64]

// CurrentSensor is a Sensor measuring the magnitude and angle of the current flow of a terminal (the from/to
// end of a branch other than Link, and a node), for use by state estimation. Generic over P (RealValue):
// SymCurrentSensor measures a single value; AsymCurrentSensor measures per-phase (a/b/c) values. See
// docs/user_manual/components.md#generic-current-sensor.
type CurrentSensor[P RealValue] struct {
	SensorBase
	// MeasuredTerminalType indicates which side of the branch is measured; must match what MeasuredObject
	// actually is.
	MeasuredTerminalType MeasuredTerminalType `json:"measured_terminal_type"`
	// AngleMeasurementType indicates whether IAngleMeasured is a global or local angle (see
	// docs/user_manual/components.md#local-angle-current-sensors).
	AngleMeasurementType AngleMeasurementType `json:"angle_measurement_type"`
	// ISigma is the standard deviation of the current magnitude measurement error, in ampere (A). Required only
	// for state estimation. Must be > 0.
	ISigma *float64 `json:"i_sigma,omitempty"`
	// IAngleSigma is the standard deviation of the current phase angle measurement error, in radian. Required
	// only for state estimation. Must be > 0.
	IAngleSigma *float64 `json:"i_angle_sigma,omitempty"`
	// IMeasured is the measured current magnitude, in ampere (A). Required only for state estimation.
	IMeasured *P `json:"i_measured,omitempty"`
	// IAngleMeasured is the measured current phase angle, in radian (meaning depends on AngleMeasurementType).
	// Required only for state estimation. Note: IMeasured=0 combined with IAngleMeasured=nπ/2 is invalid.
	IAngleMeasured *P `json:"i_angle_measured,omitempty"`
}

// SymCurrentSensor is a CurrentSensor measuring a single current value. Type name: "sym_current_sensor".
type SymCurrentSensor = CurrentSensor[float64]

// AsymCurrentSensor is a CurrentSensor measuring per-phase (a/b/c) current values. Type name:
// "asym_current_sensor".
type AsymCurrentSensor = CurrentSensor[[3]float64]
