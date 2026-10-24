package vehicle

// EngineForTeam applies the original team-selection coefficients while
// retaining the player's separately adjustable gear ratios.
func EngineForTeam(config EngineConfig, team int) EngineConfig {
	if team == 0 {
		config.Consumption, config.ShiftUp, config.ShiftDown = 1221, 4250, 1900
		config.PowerLinear, config.PowerCubic = 1986, 16224
		config.DamageRPM, config.DamageScale = 4585, 72
	} else {
		config.Consumption, config.ShiftUp, config.ShiftDown = 913, 5750, 2500
		config.PowerLinear, config.PowerCubic = 1472, 10719
		config.DamageRPM, config.DamageScale = 6000, 18
	}
	return config
}

// PlayerCarIdentity retains the original engine-sound flag in the identifier.
// The visible car number is seventeen for all three team choices.
func PlayerCarIdentity(team int) uint8 {
	if team == 0 {
		return 0x91
	}
	return 17
}

// PlayerCarColors are the original primary/secondary palette selections.
func PlayerCarColors(team int) (uint8, uint8) {
	colors := [3][2]uint8{{12, 4}, {9, 9}, {14, 14}}[team]
	return colors[0], colors[1]
}
