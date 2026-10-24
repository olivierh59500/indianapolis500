package scene

import "testing"

func TestOriginalPitSegmentContactsAndResponse(t *testing.T) {
	w := recoveredWorld(t)
	// Independent original calls to $5544/$534e, with all reference state
	// restored afterward. These cover endpoints, reversed faces, parallel motion,
	// oblique impacts and both damage-severity tables.
	tests := []struct {
		current, previous Vector2
		wall              Wall
		travel, heading   int16
		damage, hit       bool
		contact           Contact
		expected          CollisionState
		correction        Vector2
	}{
		{Vector2{9185, 6719}, Vector2{9177, 6719}, Wall{Vector2{9180, 6722}, Vector2{0, 133}}, 0, 512, true, false, Contact{}, CollisionState{}, Vector2{}},
		{Vector2{9185, 6722}, Vector2{9177, 6722}, Wall{Vector2{9180, 6722}, Vector2{0, 133}}, 0, 512, true, true, Contact{Vector2{9180, 6722}, 7, 0, -32768}, CollisionState{Position: Vector2{9178, 6722}, Velocity: Vector2{128, 0}, Heading: 513, YawVelocity: 14, TravelHeading: 0, Speed: 128, Impact: 112, PitchImpulse: 19, RollImpulse: -23, Steering: 0, DamageEnabled: true, DamageFlags: 0, FractionX: 0, FractionY: 0}, Vector2{-7, 0}},
		{Vector2{9185, 6723}, Vector2{9177, 6723}, Wall{Vector2{9180, 6722}, Vector2{0, 133}}, 0, 512, true, true, Contact{Vector2{9180, 6722}, 7, 0, -32768}, CollisionState{Position: Vector2{9178, 6723}, Velocity: Vector2{128, 0}, Heading: 513, YawVelocity: 14, TravelHeading: 0, Speed: 128, Impact: 112, PitchImpulse: 19, RollImpulse: -23, Steering: 0, DamageEnabled: true, DamageFlags: 0, FractionX: 0, FractionY: 0}, Vector2{-7, 0}},
		{Vector2{9185, 6775}, Vector2{9177, 6775}, Wall{Vector2{9180, 6722}, Vector2{0, 133}}, 0, 512, true, true, Contact{Vector2{9180, 6774}, 7, 0, -32768}, CollisionState{Position: Vector2{9178, 6775}, Velocity: Vector2{128, 0}, Heading: 513, YawVelocity: 14, TravelHeading: 0, Speed: 128, Impact: 112, PitchImpulse: 19, RollImpulse: -23, Steering: 0, DamageEnabled: true, DamageFlags: 0, FractionX: 0, FractionY: 0}, Vector2{-7, 0}},
		{Vector2{9185, 6854}, Vector2{9177, 6854}, Wall{Vector2{9180, 6722}, Vector2{0, 133}}, 0, 512, true, true, Contact{Vector2{9180, 6853}, 7, 0, -32768}, CollisionState{Position: Vector2{9178, 6854}, Velocity: Vector2{128, 0}, Heading: 513, YawVelocity: 14, TravelHeading: 0, Speed: 128, Impact: 112, PitchImpulse: 19, RollImpulse: -23, Steering: 0, DamageEnabled: true, DamageFlags: 0, FractionX: 0, FractionY: 0}, Vector2{-7, 0}},
		{Vector2{9185, 6855}, Vector2{9177, 6855}, Wall{Vector2{9180, 6722}, Vector2{0, 133}}, 0, 512, true, true, Contact{Vector2{9180, 6854}, 7, 0, -32768}, CollisionState{Position: Vector2{9178, 6855}, Velocity: Vector2{128, 0}, Heading: 513, YawVelocity: 14, TravelHeading: 0, Speed: 128, Impact: 112, PitchImpulse: 19, RollImpulse: -23, Steering: 0, DamageEnabled: true, DamageFlags: 0, FractionX: 0, FractionY: 0}, Vector2{-7, 0}},
		{Vector2{9185, 6858}, Vector2{9177, 6858}, Wall{Vector2{9180, 6722}, Vector2{0, 133}}, 0, 512, true, false, Contact{}, CollisionState{}, Vector2{}},
		{Vector2{9175, 6775}, Vector2{9185, 6775}, Wall{Vector2{9180, 6722}, Vector2{0, 133}}, -32768, -32768, false, false, Contact{}, CollisionState{}, Vector2{}},
		{Vector2{9195, 6775}, Vector2{9185, 6775}, Wall{Vector2{9180, 6722}, Vector2{0, 133}}, 0, 0, false, false, Contact{}, CollisionState{}, Vector2{}},
		{Vector2{9180, 6775}, Vector2{9180, 6765}, Wall{Vector2{9180, 6722}, Vector2{0, 133}}, -16384, -16384, false, false, Contact{}, CollisionState{}, Vector2{}},
		{Vector2{9180, 6775}, Vector2{9180, 6775}, Wall{Vector2{9180, 6722}, Vector2{0, 133}}, 0, 0, false, false, Contact{}, CollisionState{}, Vector2{}},
		{Vector2{9200, 6770}, Vector2{9160, 6770}, Wall{Vector2{9180, 6722}, Vector2{0, 133}}, 0, 0, true, true, Contact{Vector2{9180, 6769}, 22, 0, -32768}, CollisionState{Position: Vector2{9178, 6770}, Velocity: Vector2{2304, 0}, Heading: 0, YawVelocity: 13, TravelHeading: 0, Speed: 2304, Impact: 352, PitchImpulse: 19, RollImpulse: -23, Steering: 0, DamageEnabled: true, DamageFlags: 9, FractionX: 0, FractionY: 0}, Vector2{-22, 0}},
		{Vector2{9200, 6770}, Vector2{9160, 6770}, Wall{Vector2{9180, 6722}, Vector2{0, 133}}, 0, 1000, true, true, Contact{Vector2{9180, 6769}, 22, 0, -32768}, CollisionState{Position: Vector2{9178, 6770}, Velocity: Vector2{2304, 0}, Heading: 1016, YawVelocity: 29, TravelHeading: 0, Speed: 2304, Impact: 352, PitchImpulse: -77, RollImpulse: -23, Steering: 0, DamageEnabled: true, DamageFlags: 9, FractionX: 0, FractionY: 0}, Vector2{-22, 0}},
		{Vector2{9200, 6770}, Vector2{9160, 6770}, Wall{Vector2{9180, 6722}, Vector2{0, 133}}, 0, -1500, true, true, Contact{Vector2{9180, 6769}, 22, 0, -32768}, CollisionState{Position: Vector2{9178, 6770}, Velocity: Vector2{2304, 0}, Heading: -1533, YawVelocity: -20, TravelHeading: 0, Speed: 2304, Impact: 352, PitchImpulse: 163, RollImpulse: -23, Steering: 0, DamageEnabled: true, DamageFlags: 9, FractionX: 0, FractionY: 0}, Vector2{-22, 0}},
		{Vector2{9200, 6782}, Vector2{9160, 6770}, Wall{Vector2{9180, 6722}, Vector2{0, 133}}, -3040, -3040, true, true, Contact{Vector2{9180, 6775}, 22, 6, -32768}, CollisionState{Position: Vector2{9178, 6779}, Velocity: Vector2{2304, 1152}, Heading: -2698, YawVelocity: 355, TravelHeading: -4836, Speed: 2575, Impact: 400, PitchImpulse: 147, RollImpulse: -23, Steering: 0, DamageEnabled: true, DamageFlags: 9, FractionX: 0, FractionY: 0}, Vector2{-22, -3}},
		{Vector2{9200, 6782}, Vector2{9160, 6770}, Wall{Vector2{9180, 6722}, Vector2{0, 133}}, -3040, -2040, true, true, Contact{Vector2{9180, 6775}, 22, 6, -32768}, CollisionState{Position: Vector2{9178, 6779}, Velocity: Vector2{2304, 1152}, Heading: -1623, YawVelocity: 430, TravelHeading: -4836, Speed: 2575, Impact: 400, PitchImpulse: 51, RollImpulse: -23, Steering: 0, DamageEnabled: true, DamageFlags: 9, FractionX: 0, FractionY: 0}, Vector2{-22, -3}},
		{Vector2{9200, 6782}, Vector2{9160, 6770}, Wall{Vector2{9180, 6722}, Vector2{0, 133}}, -3040, -4540, true, true, Contact{Vector2{9180, 6775}, 22, 6, -32768}, CollisionState{Position: Vector2{9178, 6779}, Velocity: Vector2{2304, 1152}, Heading: -4348, YawVelocity: 205, TravelHeading: -4836, Speed: 2575, Impact: 400, PitchImpulse: 291, RollImpulse: -23, Steering: 0, DamageEnabled: true, DamageFlags: 8, FractionX: 0, FractionY: 0}, Vector2{-22, -3}},
		{Vector2{9200, 6790}, Vector2{9160, 6770}, Wall{Vector2{9180, 6722}, Vector2{0, 133}}, -4836, -4836, true, true, Contact{Vector2{9180, 6779}, 21, 9, -32768}, CollisionState{Position: Vector2{9179, 6786}, Velocity: Vector2{2432, 2048}, Heading: -4448, YawVelocity: 401, TravelHeading: -7298, Speed: 3179, Impact: 400, PitchImpulse: 131, RollImpulse: -23, Steering: 0, DamageEnabled: true, DamageFlags: 8, FractionX: 0, FractionY: 0}, Vector2{-21, -4}},
		{Vector2{9200, 6790}, Vector2{9160, 6770}, Wall{Vector2{9180, 6722}, Vector2{0, 133}}, -4836, -3836, true, true, Contact{Vector2{9180, 6779}, 21, 9, -32768}, CollisionState{Position: Vector2{9179, 6786}, Velocity: Vector2{2432, 2048}, Heading: -3334, YawVelocity: 515, TravelHeading: -7298, Speed: 3179, Impact: 400, PitchImpulse: 35, RollImpulse: -23, Steering: 0, DamageEnabled: true, DamageFlags: 9, FractionX: 0, FractionY: 0}, Vector2{-21, -4}},
		{Vector2{9200, 6790}, Vector2{9160, 6770}, Wall{Vector2{9180, 6722}, Vector2{0, 133}}, -4836, -6336, true, true, Contact{Vector2{9180, 6779}, 21, 9, -32768}, CollisionState{Position: Vector2{9179, 6786}, Velocity: Vector2{2432, 2048}, Heading: -6152, YawVelocity: 197, TravelHeading: -7298, Speed: 3179, Impact: 400, PitchImpulse: 275, RollImpulse: -23, Steering: 0, DamageEnabled: true, DamageFlags: 8, FractionX: 0, FractionY: 0}, Vector2{-21, -4}},
		{Vector2{9200, 6798}, Vector2{9160, 6770}, Wall{Vector2{9180, 6722}, Vector2{0, 133}}, -6370, -6370, true, true, Contact{Vector2{9180, 6783}, 22, 14, -32768}, CollisionState{Position: Vector2{9178, 6791}, Velocity: Vector2{2304, 2688}, Heading: -5867, YawVelocity: 516, TravelHeading: -8994, Speed: 3540, Impact: 464, PitchImpulse: 139, RollImpulse: -23, Steering: 0, DamageEnabled: true, DamageFlags: 8, FractionX: 0, FractionY: 0}, Vector2{-22, -7}},
		{Vector2{9200, 6798}, Vector2{9160, 6770}, Wall{Vector2{9180, 6722}, Vector2{0, 133}}, -6370, -5370, true, true, Contact{Vector2{9180, 6783}, 22, 14, -32768}, CollisionState{Position: Vector2{9178, 6791}, Velocity: Vector2{2304, 2688}, Heading: -4708, YawVelocity: 675, TravelHeading: -8994, Speed: 3540, Impact: 464, PitchImpulse: 43, RollImpulse: -23, Steering: 0, DamageEnabled: true, DamageFlags: 8, FractionX: 0, FractionY: 0}, Vector2{-22, -7}},
		{Vector2{9200, 6798}, Vector2{9160, 6770}, Wall{Vector2{9180, 6722}, Vector2{0, 133}}, -6370, -7870, true, true, Contact{Vector2{9180, 6783}, 22, 14, -32768}, CollisionState{Position: Vector2{9178, 6791}, Velocity: Vector2{2304, 2688}, Heading: -7628, YawVelocity: 255, TravelHeading: -8994, Speed: 3540, Impact: 464, PitchImpulse: 283, RollImpulse: -23, Steering: 0, DamageEnabled: true, DamageFlags: 8, FractionX: 0, FractionY: 0}, Vector2{-22, -7}},
		{Vector2{9200, 6810}, Vector2{9160, 6770}, Wall{Vector2{9180, 6722}, Vector2{0, 133}}, -8192, -8192, true, true, Contact{Vector2{9180, 6789}, 22, 20, -32768}, CollisionState{Position: Vector2{9178, 6800}, Velocity: Vector2{2304, 3840}, Heading: -7616, YawVelocity: 589, TravelHeading: -10750, Speed: 4478, Impact: 512, PitchImpulse: 19, RollImpulse: -23, Steering: 0, DamageEnabled: true, DamageFlags: 8, FractionX: 0, FractionY: 0}, Vector2{-22, -10}},
		{Vector2{9200, 6810}, Vector2{9160, 6770}, Wall{Vector2{9180, 6722}, Vector2{0, 133}}, -8192, -7192, true, true, Contact{Vector2{9180, 6789}, 22, 20, -32768}, CollisionState{Position: Vector2{9178, 6800}, Velocity: Vector2{2304, 3840}, Heading: -6424, YawVelocity: 781, TravelHeading: -10750, Speed: 4478, Impact: 512, PitchImpulse: -77, RollImpulse: -23, Steering: 0, DamageEnabled: true, DamageFlags: 8, FractionX: 0, FractionY: 0}, Vector2{-22, -10}},
		{Vector2{9200, 6810}, Vector2{9160, 6770}, Wall{Vector2{9180, 6722}, Vector2{0, 133}}, -8192, -9692, true, true, Contact{Vector2{9180, 6789}, 22, 20, -32768}, CollisionState{Position: Vector2{9178, 6800}, Velocity: Vector2{2304, 3840}, Heading: -9413, YawVelocity: 292, TravelHeading: -10750, Speed: 4478, Impact: 512, PitchImpulse: 163, RollImpulse: -23, Steering: 0, DamageEnabled: true, DamageFlags: 8, FractionX: 0, FractionY: 0}, Vector2{-22, -10}},
		{Vector2{-4250, 6994}, Vector2{-4256, 7010}, Wall{Vector2{6520, 7002}, Vector2{-10780, 0}}, 12642, 13342, true, false, Contact{}, CollisionState{}, Vector2{}},
		{Vector2{-4250, 7010}, Vector2{-4250, 6994}, Wall{Vector2{6520, 7002}, Vector2{-10780, 0}}, -16384, -16384, false, true, Contact{Vector2{-4250, 7002}, 10, 0, 16384}, CollisionState{Position: Vector2{-4250, 7000}, Velocity: Vector2{0, 768}, Heading: -16384, YawVelocity: 13, TravelHeading: -16384, Speed: 768, Impact: 160, PitchImpulse: 19, RollImpulse: -23, Steering: 0, DamageEnabled: false, DamageFlags: 0, FractionX: 0, FractionY: 0}, Vector2{0, -10}},
		{Vector2{0, 6994}, Vector2{-6, 7010}, Wall{Vector2{6520, 7002}, Vector2{-10780, 0}}, 12642, 13342, true, false, Contact{}, CollisionState{}, Vector2{}},
		{Vector2{0, 7010}, Vector2{0, 6994}, Wall{Vector2{6520, 7002}, Vector2{-10780, 0}}, -16384, -16384, false, true, Contact{Vector2{0, 7002}, 10, 0, 16384}, CollisionState{Position: Vector2{0, 7000}, Velocity: Vector2{0, 768}, Heading: -16384, YawVelocity: 13, TravelHeading: -16384, Speed: 768, Impact: 160, PitchImpulse: 19, RollImpulse: -23, Steering: 0, DamageEnabled: false, DamageFlags: 0, FractionX: 0, FractionY: 0}, Vector2{0, -10}},
		{Vector2{4100, 6994}, Vector2{4094, 7010}, Wall{Vector2{6520, 7002}, Vector2{-10780, 0}}, 12642, 13342, true, false, Contact{}, CollisionState{}, Vector2{}},
		{Vector2{4100, 7010}, Vector2{4100, 6994}, Wall{Vector2{6520, 7002}, Vector2{-10780, 0}}, -16384, -16384, false, true, Contact{Vector2{4100, 7002}, 10, 0, 16384}, CollisionState{Position: Vector2{4100, 7000}, Velocity: Vector2{0, 768}, Heading: -16384, YawVelocity: 13, TravelHeading: -16384, Speed: 768, Impact: 160, PitchImpulse: 19, RollImpulse: -23, Steering: 0, DamageEnabled: false, DamageFlags: 0, FractionX: 0, FractionY: 0}, Vector2{0, -10}},
		{Vector2{6500, 6994}, Vector2{6494, 7010}, Wall{Vector2{6520, 7002}, Vector2{-10780, 0}}, 12642, 13342, true, false, Contact{}, CollisionState{}, Vector2{}},
		{Vector2{6500, 7010}, Vector2{6500, 6994}, Wall{Vector2{6520, 7002}, Vector2{-10780, 0}}, -16384, -16384, false, true, Contact{Vector2{6500, 7002}, 10, 0, 16384}, CollisionState{Position: Vector2{6500, 7000}, Velocity: Vector2{0, 768}, Heading: -16384, YawVelocity: 13, TravelHeading: -16384, Speed: 768, Impact: 160, PitchImpulse: 19, RollImpulse: -23, Steering: 0, DamageEnabled: false, DamageFlags: 0, FractionX: 0, FractionY: 0}, Vector2{0, -10}},
		{Vector2{6518, 6994}, Vector2{6512, 7010}, Wall{Vector2{6520, 7002}, Vector2{-10780, 0}}, 12642, 13342, true, false, Contact{}, CollisionState{}, Vector2{}},
		{Vector2{6518, 7010}, Vector2{6518, 6994}, Wall{Vector2{6520, 7002}, Vector2{-10780, 0}}, -16384, -16384, false, true, Contact{Vector2{6518, 7002}, 10, 0, 16384}, CollisionState{Position: Vector2{6518, 7000}, Velocity: Vector2{0, 768}, Heading: -16384, YawVelocity: 13, TravelHeading: -16384, Speed: 768, Impact: 160, PitchImpulse: 19, RollImpulse: -23, Steering: 0, DamageEnabled: false, DamageFlags: 0, FractionX: 0, FractionY: 0}, Vector2{0, -10}},
		{Vector2{9200, 6780}, Vector2{9160, 6770}, Wall{Vector2{9180, 6722}, Vector2{0, 133}}, -2555, -2555, true, true, Contact{Vector2{9180, 6774}, 21, 4, -32768}, CollisionState{Position: Vector2{9179, 6778}, Velocity: Vector2{2432, 1024}, Heading: -2327, YawVelocity: 241, TravelHeading: -4155, Speed: 2638, Impact: 368, PitchImpulse: 39, RollImpulse: -23, Steering: 0, DamageEnabled: true, DamageFlags: 9, FractionX: 0, FractionY: 0}, Vector2{-21, -2}},
		{Vector2{9200, 6780}, Vector2{9160, 6770}, Wall{Vector2{9180, 6722}, Vector2{0, 133}}, -2555, -507, true, true, Contact{Vector2{9180, 6774}, 21, 4, -32768}, CollisionState{Position: Vector2{9179, 6778}, Velocity: Vector2{2432, 1024}, Heading: -194, YawVelocity: 326, TravelHeading: -4155, Speed: 2638, Impact: 368, PitchImpulse: 39, RollImpulse: -23, Steering: 0, DamageEnabled: true, DamageFlags: 9, FractionX: 0, FractionY: 0}, Vector2{-21, -2}},
		{Vector2{9200, 6780}, Vector2{9160, 6770}, Wall{Vector2{9180, 6722}, Vector2{0, 133}}, -2555, -12555, true, true, Contact{Vector2{9180, 6774}, 21, 4, -32768}, CollisionState{Position: Vector2{9179, 6778}, Velocity: Vector2{2432, 1024}, Heading: -13518, YawVelocity: -950, TravelHeading: -4155, Speed: 2638, Impact: 368, PitchImpulse: -25, RollImpulse: -23, Steering: 0, DamageEnabled: true, DamageFlags: 12, FractionX: 0, FractionY: 0}, Vector2{-21, -2}},
		{Vector2{9230, 6780}, Vector2{9160, 6770}, Wall{Vector2{9180, 6722}, Vector2{0, 133}}, -1480, -1480, true, true, Contact{Vector2{9180, 6772}, 51, 7, -32768}, CollisionState{Position: Vector2{9179, 6777}, Velocity: Vector2{2432, 896}, Heading: -1003, YawVelocity: 490, TravelHeading: -3679, Speed: 2591, Impact: 864, PitchImpulse: 243, RollImpulse: -23, Steering: 0, DamageEnabled: true, DamageFlags: 25, FractionX: 0, FractionY: 0}, Vector2{-51, -3}},
		{Vector2{9230, 6780}, Vector2{9160, 6770}, Wall{Vector2{9180, 6722}, Vector2{0, 133}}, -1480, 568, true, true, Contact{Vector2{9180, 6772}, 51, 7, -32768}, CollisionState{Position: Vector2{9179, 6777}, Velocity: Vector2{2432, 896}, Heading: 1139, YawVelocity: 584, TravelHeading: -3679, Speed: 2591, Impact: 864, PitchImpulse: 243, RollImpulse: -23, Steering: 0, DamageEnabled: true, DamageFlags: 25, FractionX: 0, FractionY: 0}, Vector2{-51, -3}},
		{Vector2{9230, 6780}, Vector2{9160, 6770}, Wall{Vector2{9180, 6722}, Vector2{0, 133}}, -1480, -11480, true, true, Contact{Vector2{9180, 6772}, 51, 7, -32768}, CollisionState{Position: Vector2{9179, 6777}, Velocity: Vector2{2432, 896}, Heading: -13700, YawVelocity: -2207, TravelHeading: -3679, Speed: 2591, Impact: 864, PitchImpulse: 179, RollImpulse: -23, Steering: 0, DamageEnabled: true, DamageFlags: 28, FractionX: 0, FractionY: 0}, Vector2{-51, -3}},
		{Vector2{9300, 6780}, Vector2{9160, 6770}, Wall{Vector2{9180, 6722}, Vector2{0, 133}}, -744, -744, true, true, Contact{Vector2{9180, 6771}, 121, 8, -32768}, CollisionState{Position: Vector2{9179, 6776}, Velocity: Vector2{2432, 768}, Heading: -167, YawVelocity: 590, TravelHeading: -3187, Speed: 2550, Impact: 2000, PitchImpulse: 115, RollImpulse: -23, Steering: 0, DamageEnabled: true, DamageFlags: 25, FractionX: 0, FractionY: 0}, Vector2{-121, -4}},
		{Vector2{9300, 6780}, Vector2{9160, 6770}, Wall{Vector2{9180, 6722}, Vector2{0, 133}}, -744, 1304, true, true, Contact{Vector2{9180, 6771}, 121, 8, -32768}, CollisionState{Position: Vector2{9179, 6776}, Velocity: Vector2{2432, 768}, Heading: 2090, YawVelocity: 799, TravelHeading: -3187, Speed: 2550, Impact: 2000, PitchImpulse: 115, RollImpulse: -23, Steering: 0, DamageEnabled: true, DamageFlags: 25, FractionX: 0, FractionY: 0}, Vector2{-121, -4}},
		{Vector2{9300, 6780}, Vector2{9160, 6770}, Wall{Vector2{9180, 6722}, Vector2{0, 133}}, -744, -10744, true, true, Contact{Vector2{9180, 6771}, 121, 8, -32768}, CollisionState{Position: Vector2{9179, 6776}, Velocity: Vector2{2432, 768}, Heading: -7819, YawVelocity: 2938, TravelHeading: -3187, Speed: 2550, Impact: 2000, PitchImpulse: 51, RollImpulse: -23, Steering: 0, DamageEnabled: true, DamageFlags: 28, FractionX: 0, FractionY: 0}, Vector2{-121, -4}},
	}
	for i, test := range tests {
		contact, hit := w.WallContact(test.current, test.previous, test.travel, test.wall)
		if hit != test.hit {
			t.Fatalf("original contact%d hit=%v, want %v", i, hit, test.hit)
		}
		if !hit {
			continue
		}
		if contact != test.contact {
			t.Fatalf("original contact%d got%+v want%+v", i, contact, test.contact)
		}
		state := CollisionState{
			Position: test.current, Velocity: Vector2{(test.current.X - test.previous.X) * 128, (test.current.Y - test.previous.Y) * 128},
			FractionX: 0x1234, FractionY: 0x5678, Heading: test.heading, YawVelocity: 13, TravelHeading: test.travel,
			Speed: 777, Impact: 64, PitchImpulse: 19, RollImpulse: -23, Steering: 512, DamageEnabled: test.damage,
		}
		correction := w.ApplyContact(&state, contact)
		if correction != test.correction || state != test.expected {
			t.Fatalf("original response%d got%+v correction%+v, want%+v correction%+v", i, state, correction, test.expected, test.correction)
		}
	}
}

func TestOriginalPitBarrierSelection(t *testing.T) {
	w := recoveredWorld(t)
	tests := []struct {
		p   uint16
		lat int16
		set int
	}{
		{7005, 39, 1}, {7005, 400, 0}, {7999, 39, 1}, {8000, 39, 2},
		{8399, 39, 2}, {8400, 39, 3}, {9399, 39, 3}, {9400, 39, 4},
		{9699, 39, 4}, {9700, 39, -1}, {39399, 39, -1}, {39400, 39, 1},
		{39400, 300, -1}, {39599, 39, 1},
	}
	for _, test := range tests {
		walls := w.PitBarriers(test.p, test.lat)
		if test.set < 0 {
			if len(walls) != 0 {
				t.Fatal("a pit barrier appeared outside its source section")
			}
			continue
		}
		if len(walls) != len(w.barriers[test.set]) || &walls[0] != &w.barriers[test.set][0] {
			t.Fatalf("progress%d lateral%d selected the wrong original barrier set", test.p, test.lat)
		}
	}
}

func TestOriginalWorldToTrackLocation(t *testing.T) {
	w := recoveredWorld(t)
	// Captured from the original inverse-position block before wall handling.
	tests := []struct {
		point    Vector2
		location TrackLocation
	}{
		{Vector2{4110, 6761}, TrackLocation{7005, 39, 20, 580}},
		{Vector2{4110, 7022}, TrackLocation{7005, 300, 20, 580}},
		{Vector2{4110, 7222}, TrackLocation{7005, 500, 20, 580}},
		{Vector2{4444, 6761}, TrackLocation{7172, 39, 20, 580}},
		{Vector2{4444, 7022}, TrackLocation{7172, 300, 20, 580}},
		{Vector2{4444, 7222}, TrackLocation{7172, 500, 20, 580}},
		{Vector2{10089, 6757}, TrackLocation{9999, 38, -12, 584}},
		{Vector2{10099, 7017}, TrackLocation{9998, 298, -12, 584}},
		{Vector2{10107, 7217}, TrackLocation{9998, 499, -12, 584}},
		{Vector2{13280, 5360}, TrackLocation{11880, 38, 19, 746}},
		{Vector2{13465, 5545}, TrackLocation{11880, 299, 19, 746}},
		{Vector2{13606, 5686}, TrackLocation{11880, 499, 19, 746}},
		{Vector2{14681, -20}, TrackLocation{14860, 39, 20, 640}},
		{Vector2{14942, -20}, TrackLocation{14860, 300, 20, 640}},
		{Vector2{15142, -20}, TrackLocation{14860, 500, 20, 640}},
		{Vector2{13280, -5360}, TrackLocation{17820, 38, 19, 746}},
		{Vector2{13465, -5545}, TrackLocation{17820, 299, 19, 746}},
		{Vector2{13606, -5686}, TrackLocation{17820, 499, 19, 746}},
		{Vector2{5500, -6761}, TrackLocation{22000, 39, 20, 580}},
		{Vector2{5500, -7022}, TrackLocation{22000, 300, 20, 580}},
		{Vector2{5500, -7222}, TrackLocation{22000, 500, 20, 580}},
		{Vector2{-13562, -5053}, TrackLocation{31900, 38, -43, 738}},
		{Vector2{-13762, -5220}, TrackLocation{31900, 299, -43, 738}},
		{Vector2{-13915, -5349}, TrackLocation{31900, 499, -43, 738}},
		{Vector2{-14681, -180}, TrackLocation{34560, 39, 20, 640}},
		{Vector2{-14942, -180}, TrackLocation{34560, 300, 20, 640}},
		{Vector2{-15142, -180}, TrackLocation{34560, 500, 20, 640}},
		{Vector2{-13280, 5360}, TrackLocation{37620, 38, 19, 746}},
		{Vector2{-13465, 5545}, TrackLocation{37620, 299, 19, 746}},
		{Vector2{-13606, 5686}, TrackLocation{37620, 499, 19, 746}},
		{Vector2{9178, 6775}, TrackLocation{9539, 53, 20, 580}},
		{Vector2{9178, 6779}, TrackLocation{9539, 57, 20, 580}},
		{Vector2{9178, 6800}, TrackLocation{9539, 78, 20, 580}},
		{Vector2{6500, 7000}, TrackLocation{8200, 278, 20, 580}},
		{Vector2{-9900, 7000}, TrackLocation{0, 278, 20, 580}},
		{Vector2{9900, 7000}, TrackLocation{9900, 278, 20, 580}},
		{Vector2{15092, 1980}, TrackLocation{13860, 450, 20, 640}},
		{Vector2{-15092, -1980}, TrackLocation{33660, 450, 20, 640}},
	}
	for _, test := range tests {
		if got := w.LocateTrack(test.point); got != test.location {
			t.Fatalf("original inverse %+v got%+v want%+v", test.point, got, test.location)
		}
	}
}

func TestOriginalRadialWallContacts(t *testing.T) {
	w := recoveredWorld(t)
	tests := []struct {
		previous, current TrackLocation
		heading           int16
		contact           Contact
	}{
		{TrackLocation{7005, 39, 20, 580}, TrackLocation{7010, 10, 20, 580}, 0, Contact{Depth: 12, Tangent: 4, Normal: -16384}},
		{TrackLocation{7005, 39, 20, 580}, TrackLocation{7010, 10, 20, 580}, 8464, Contact{Depth: 12, Tangent: 4, Normal: -7920}},
		{TrackLocation{7005, 21, 20, 580}, TrackLocation{7010, 19, 20, 580}, 0, Contact{Depth: 3, Tangent: 30, Normal: -16384}},
		{TrackLocation{7005, 21, 20, 580}, TrackLocation{7010, 19, 20, 580}, 8464, Contact{Depth: 3, Tangent: 30, Normal: -7920}},
		{TrackLocation{7005, 579, 20, 580}, TrackLocation{7010, 581, 20, 580}, 0, Contact{Depth: 3, Tangent: -30, Normal: 16384}},
		{TrackLocation{7005, 579, 20, 580}, TrackLocation{7010, 581, 20, 580}, 8464, Contact{Depth: 3, Tangent: -30, Normal: 24848}},
		{TrackLocation{7005, 560, 20, 580}, TrackLocation{7010, 610, 20, 580}, 0, Contact{Depth: 32, Tangent: -6, Normal: 16384}},
		{TrackLocation{7005, 560, 20, 580}, TrackLocation{7010, 610, 20, 580}, 8464, Contact{Depth: 32, Tangent: -6, Normal: 24848}},
		{TrackLocation{7010, 39, 20, 580}, TrackLocation{7005, 10, 20, 580}, 0, Contact{Depth: 12, Tangent: -4, Normal: -16384}},
		{TrackLocation{7010, 39, 20, 580}, TrackLocation{7005, 10, 20, 580}, 8464, Contact{Depth: 12, Tangent: -4, Normal: -7920}},
		{TrackLocation{7010, 21, 20, 580}, TrackLocation{7005, 19, 20, 580}, 0, Contact{Depth: 3, Tangent: -30, Normal: -16384}},
		{TrackLocation{7010, 21, 20, 580}, TrackLocation{7005, 19, 20, 580}, 8464, Contact{Depth: 3, Tangent: -30, Normal: -7920}},
		{TrackLocation{7010, 579, 20, 580}, TrackLocation{7005, 581, 20, 580}, 0, Contact{Depth: 3, Tangent: 30, Normal: 16384}},
		{TrackLocation{7010, 579, 20, 580}, TrackLocation{7005, 581, 20, 580}, 8464, Contact{Depth: 3, Tangent: 30, Normal: 24848}},
		{TrackLocation{7010, 560, 20, 580}, TrackLocation{7005, 610, 20, 580}, 0, Contact{Depth: 32, Tangent: 6, Normal: 16384}},
		{TrackLocation{7010, 560, 20, 580}, TrackLocation{7005, 610, 20, 580}, 8464, Contact{Depth: 32, Tangent: 6, Normal: 24848}},
		{TrackLocation{39590, 39, 20, 580}, TrackLocation{30, 10, 20, 580}, 0, Contact{Depth: 12, Tangent: 33, Normal: -16384}},
		{TrackLocation{39590, 39, 20, 580}, TrackLocation{30, 10, 20, 580}, 8464, Contact{Depth: 12, Tangent: 33, Normal: -7920}},
		{TrackLocation{39590, 21, 20, 580}, TrackLocation{30, 19, 20, 580}, 0, Contact{Depth: 3, Tangent: 240, Normal: -16384}},
		{TrackLocation{39590, 21, 20, 580}, TrackLocation{30, 19, 20, 580}, 8464, Contact{Depth: 3, Tangent: 240, Normal: -7920}},
		{TrackLocation{39590, 579, 20, 580}, TrackLocation{30, 581, 20, 580}, 0, Contact{Depth: 3, Tangent: -240, Normal: 16384}},
		{TrackLocation{39590, 579, 20, 580}, TrackLocation{30, 581, 20, 580}, 8464, Contact{Depth: 3, Tangent: -240, Normal: 24848}},
		{TrackLocation{39590, 560, 20, 580}, TrackLocation{30, 610, 20, 580}, 0, Contact{Depth: 32, Tangent: -51, Normal: 16384}},
		{TrackLocation{39590, 560, 20, 580}, TrackLocation{30, 610, 20, 580}, 8464, Contact{Depth: 32, Tangent: -51, Normal: 24848}},
		{TrackLocation{10, 39, 20, 580}, TrackLocation{39580, 10, 20, 580}, 0, Contact{Depth: 12, Tangent: -24, Normal: -16384}},
		{TrackLocation{10, 39, 20, 580}, TrackLocation{39580, 10, 20, 580}, 8464, Contact{Depth: 12, Tangent: -24, Normal: -7920}},
		{TrackLocation{10, 21, 20, 580}, TrackLocation{39580, 19, 20, 580}, 0, Contact{Depth: 3, Tangent: -180, Normal: -16384}},
		{TrackLocation{10, 21, 20, 580}, TrackLocation{39580, 19, 20, 580}, 8464, Contact{Depth: 3, Tangent: -180, Normal: -7920}},
		{TrackLocation{10, 579, 20, 580}, TrackLocation{39580, 581, 20, 580}, 0, Contact{Depth: 3, Tangent: 180, Normal: 16384}},
		{TrackLocation{10, 579, 20, 580}, TrackLocation{39580, 581, 20, 580}, 8464, Contact{Depth: 3, Tangent: 180, Normal: 24848}},
		{TrackLocation{10, 560, 20, 580}, TrackLocation{39580, 610, 20, 580}, 0, Contact{Depth: 32, Tangent: 38, Normal: 16384}},
		{TrackLocation{10, 560, 20, 580}, TrackLocation{39580, 610, 20, 580}, 8464, Contact{Depth: 32, Tangent: 38, Normal: 24848}},
	}
	for i, test := range tests {
		got, hit := w.SurfaceContact(test.previous, test.current, test.heading)
		if !hit || got != test.contact {
			t.Fatalf("source radial contact%d got%+v/%v want%+v", i, got, hit, test.contact)
		}
	}
}

func TestOriginalCompleteWallResolution(t *testing.T) {
	w := recoveredWorld(t)
	// Original $3d64-$40cc, including repeated intersections. The two-pixel
	// tolerance may produce another contact after the first correction.
	tests := []struct {
		previous, current Vector2
		travel, heading   int16
		expected          CollisionState
		location          TrackLocation
	}{
		{Vector2{9177, 6775}, Vector2{9185, 6775}, 0, 512, CollisionState{Position: Vector2{9174, 6775}, Velocity: Vector2{-384, 0}, Heading: 514, YawVelocity: 15, TravelHeading: -32768, Speed: 384, Impact: 112, PitchImpulse: 23, RollImpulse: -23, Steering: 0, FractionX: 0, FractionY: 0, DamageFlags: 0}, TrackLocation{9537, 53, 20, 580}},
		{Vector2{9177, 6775}, Vector2{9179, 6775}, 0, 512, CollisionState{Position: Vector2{9176, 6775}, Velocity: Vector2{-128, 0}, Heading: 512, YawVelocity: 13, TravelHeading: -32768, Speed: 128, Impact: 48, PitchImpulse: 19, RollImpulse: -23, Steering: 0, FractionX: 0, FractionY: 0, DamageFlags: 0}, TrackLocation{9538, 53, 20, 580}},
		{Vector2{9160, 6770}, Vector2{9200, 6790}, -4836, -4324, CollisionState{Position: Vector2{9177, 6786}, Velocity: Vector2{2176, 2048}, Heading: -3890, YawVelocity: 447, TravelHeading: -7875, Speed: 2988, Impact: 400, PitchImpulse: -1, RollImpulse: -23, Steering: 0, FractionX: 0, FractionY: 0, DamageFlags: 0}, TrackLocation{9538, 64, 20, 580}},
		{Vector2{0, 6990}, Vector2{6, 7026}, -14661, -14149, CollisionState{Position: Vector2{3, 6999}, Velocity: Vector2{384, 1152}, Heading: -14462, YawVelocity: -300, TravelHeading: -13031, Speed: 1214, Impact: 432, PitchImpulse: -465, RollImpulse: -23, Steering: 0, FractionX: 0, FractionY: 0, DamageFlags: 0}, TrackLocation{4951, 277, 20, 580}},
		{Vector2{4090, 6761}, Vector2{4110, 6732}, 10087, 10599, CollisionState{Position: Vector2{4106, 6744}, Velocity: Vector2{2048, -2176}, Heading: 10951, YawVelocity: 365, TravelHeading: 8509, Speed: 2988, Impact: 256, PitchImpulse: 431, RollImpulse: -23, Steering: 0, FractionX: 0, FractionY: 0, DamageFlags: 0}, TrackLocation{7003, 22, 20, 580}},
		{Vector2{4090, 7290}, Vector2{4110, 7310}, -8192, -7680, CollisionState{Position: Vector2{4105, 7300}, Velocity: Vector2{1920, 1280}, Heading: -7952, YawVelocity: -259, TravelHeading: -6132, Speed: 2307, Impact: 240, PitchImpulse: 19, RollImpulse: -23, Steering: 0, FractionX: 0, FractionY: 0, DamageFlags: 0}, TrackLocation{7002, 578, 20, 580}},
	}
	for i, test := range tests {
		state := CollisionState{Position: test.current, Velocity: Vector2{(test.current.X - test.previous.X) * 128, (test.current.Y - test.previous.Y) * 128}, FractionX: 0x1234, FractionY: 0x5678, Heading: test.heading, YawVelocity: 13, TravelHeading: test.travel, Speed: 777, PitchImpulse: 19, RollImpulse: -23, Steering: 512}
		location, hit := w.ResolveWalls(&state, test.previous, w.LocateTrack(test.previous), 0)
		if !hit || location != test.location || state != test.expected {
			t.Fatalf("original complete impact%d got%+v %+v/%v, want%+v %+v", i, state, location, hit, test.expected, test.location)
		}
	}
}
