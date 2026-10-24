package vehicle

import (
	"encoding/binary"
	"fmt"
)

// ContactCurve is the recovered, symmetric slip-angle response table.
// Its angle argument uses 65536 units per turn and folds into a quarter turn.
type ContactCurve [66]int16

func NewContactCurve(data []byte) (ContactCurve, error) {
	var curve ContactCurve
	if len(data) != len(curve)*2 {
		return curve, fmt.Errorf("recovered tire slip curve has invalid dimensions")
	}
	for i := range curve {
		curve[i] = int16(binary.BigEndian.Uint16(data[i*2:]))
	}
	return curve, nil
}

type WheelMotion struct{ Forward, Sideways int16 }
type ContactDemand struct {
	Longitudinal, Lateral uint16
	Skid                  byte
}

func velocityDemand(velocity int16, normalization uint16) uint16 {
	value := (uint32(unsignedAbs(velocity)*2) * uint32(normalization)) >> 8
	return uint16(min(uint32(65535), value))
}

// Demand translates the slip law at $4fe8 after the original polar velocity
// calculation. The curve is interpolated with the same signed word rounding;
// longitudinal and lateral demands remain unsigned traction coefficients.
func (curve *ContactCurve) Demand(motion WheelMotion, angle int16, grip, normalization uint16) ContactDemand {
	longitudinal := min(grip, velocityDemand(motion.Forward, normalization))
	lateralLimit := velocityDemand(motion.Sideways, normalization)
	folded := unsignedAbs(angle)
	if folded > 16383 {
		folded = 32768 - folded
	}
	skid := byte(0)
	if folded >= 500 {
		skid = 127
		if folded-500 < 1270 {
			skid = byte((folded - 500) / 10)
		}
	}
	index := folded >> 8
	base := curve[index]
	difference := int16(curve[index+1] - base)
	interpolated := base + int16(int32(int16(folded&255))*int32(difference)>>8)
	lateral := min(lateralLimit, highProduct(uint16(interpolated), grip))
	return ContactDemand{longitudinal, lateral, skid}
}
