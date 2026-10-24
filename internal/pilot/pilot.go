package pilot

import (
	"math"

	worldscene "github.com/olivierh59500/indianapolis500/internal/scene"
	"github.com/olivierh59500/indianapolis500/internal/traffic"
	"github.com/olivierh59500/indianapolis500/internal/vehicle"
)

const (
	trackLength        = 39600
	progressPerMPHTick = 44.0 / 150
	speedPerMPH        = 128 * 44.0 / 75
)

// Frame is the driver's current observation, with original setup coefficients
// and driver ramps. Traffic is absent during an empty qualifying session.
type Frame struct {
	State                  vehicle.State
	Config                 vehicle.Config
	Driver                 vehicle.DriverState
	Cars                   [33]traffic.Car
	Tick                   int
	Traffic, DamageEnabled bool
}

// New starts a driver for a fresh event without changing the car or its setup.
func New(world *worldscene.World, curve *vehicle.ContactCurve, surface *vehicle.TrackSurface) *Pilot {
	return &Pilot{world: world, curve: curve, surface: surface, maxSpeed: 240, steerLook: 12, line: 5200, minLane: 5200, sideGap: 1000, longGap: 140, followGap: 80, horizon: 18}
}

// Pilot chooses ordinary key intents using immutable observations and local
// copies of the shared force model. Predicted motion never replaces live state.
type Pilot struct {
	world                                *worldscene.World
	curve                                *vehicle.ContactCurve
	surface                              *vehicle.TrackSurface
	maxSpeed, line, lateral, laneTarget  float64
	steerLook                            float64
	minLane, sideGap, longGap, followGap float64
	horizon                              int

	leavingPit, recoveringBrake bool
	reverseDirection            int
}

// Only ordinary steering and pedal inputs enter the unchanged shared model.
func (p *Pilot) baseline(f Frame) vehicle.DriverIntent {
	s := f.State
	track := p.world.TrackPlacement(s.Progress, s.Lateral<<4)
	forward, _ := p.world.VelocityOnTrack(worldscene.Vector2{X: s.Body.VelocityX, Y: s.Body.VelocityY}, track.Heading)
	speed := float64(forward) / (128 * 44.0 / 75)
	targetLateral := p.line
	trafficLimit := p.maxSpeed

	if f.Traffic {

		inner, outer := p.world.TrackLimits(uint16((int(s.Progress) + int(max(speed, 40)*(44.0/150)*8)) % 39600))
		low, high := max(p.minLane, float64(inner+48)*16), float64(outer-48)*16
		actual := float64(s.Lateral) * 16
		if p.laneTarget == 0 {
			p.laneTarget = actual
		}
		baseLine := p.line
		if f.Tick < 150 {
			baseLine = actual
		}
		candidates := []float64{p.laneTarget, actual, baseLine, 4400, 5200, 6400, 7600, 8800}
		bestCost := math.Inf(1)
		for _, candidate := range candidates {
			candidate = max(low, min(high, candidate))
			cost := math.Abs(candidate-baseLine)/1800 + math.Abs(candidate-p.laneTarget)/650
			for i := 1; i < len(f.Cars); i++ {
				other := f.Cars[i]
				if other.Next == i {
					continue
				}
				gap := float64(int(f.Cars[0].Distance) - int(other.Distance))
				if gap > 19800 {
					gap -= 39600
				}
				if gap < -19800 {
					gap += 39600
				}
				relative := float64(int(other.Speed)-int(f.Cars[0].Speed)) / 256
				for _, future := range []float64{0, 4, 8, 12, 18, 24} {
					along := gap + relative*future
					if math.Abs(along) > p.longGap {
						continue
					}
					otherLane := float64(other.Lateral) + max(-100, min(100, float64(other.LateralVelocity)))*min(future, 8)
					ourLane := actual + (candidate-actual)*min(1, future/20)
					clearance := math.Abs(ourLane - otherLane)
					if clearance < p.sideGap {
						cost += (p.sideGap - clearance) / 10 * (1 - math.Abs(along)/p.longGap) / (1 + future/12)
					}
				}
			}
			if cost < bestCost {
				bestCost = cost
				targetLateral = candidate
			}
		}
		p.laneTarget = targetLateral
		// Keep a braking margin when a lane cannot clear a closing car in time.
		for i := 1; i < len(f.Cars); i++ {
			other := f.Cars[i]
			if other.Next == i {
				continue
			}
			gap := float64(int(f.Cars[0].Distance) - int(other.Distance))
			if gap > 19800 {
				gap -= 39600
			}
			if gap < -19800 {
				gap += 39600
			}
			if gap <= 0 || gap > p.longGap*4 {
				continue
			}
			if math.Abs(actual-float64(other.Lateral)) > p.sideGap && math.Abs(targetLateral-float64(other.Lateral)) > p.sideGap {
				continue
			}
			otherMPH := float64(other.Speed) / (128 * 44.0 / 75)
			trafficLimit = min(trafficLimit, max(40, otherMPH+(gap-p.followGap)/12))
		}
	}

	// Leave the real pit corridor before joining the banked racing surface.
	if p.lateral == 0 {
		p.leavingPit = s.Lateral < 180 && s.Progress >= 6930 && s.Progress < 9600
	}
	if p.leavingPit && s.Progress >= 9700 {
		p.leavingPit = false
	}
	if p.leavingPit {
		targetLateral = 2400
		if s.Progress >= 8400 {
			targetLateral = 3520
		}
	}
	p.laneTarget = targetLateral
	if p.lateral == 0 {
		p.lateral = float64(s.Lateral) * 16
	}
	p.lateral += max(-100, min(100, targetLateral-p.lateral))
	look := p.steerLook
	if look == 0 {
		look = 8
	}
	lookahead := float64(s.Progress) + max(speed, 40)*(44.0/150)*look
	target := p.world.TrackPlacement(uint16(int(lookahead)%39600), int16(p.lateral))
	heading := p.world.Direction(target.X-int16(s.Body.PositionX>>16), target.Y-int16(s.Body.PositionY>>16))
	desired := max(-3200, min(3200, float64(int16(heading-s.Body.Heading))*.7-float64(s.Body.YawVelocity)*.5+float64(int16(s.Body.TravelHeading-s.Body.Heading))*.5))
	steer := 0
	best := math.Inf(1)
	for _, i := range []int{0, -1, 1} {
		d := vehicle.Driver{State: f.Driver}
		c := d.Step(vehicle.DriverContext{Steering: s.Steering, Damping: s.SteeringDamping, Increment: s.SteeringIncrement, Speed: s.Body.Speed, Failures: s.Engine.Failures}, vehicle.DriverIntent{Steering: float64(i)})
		cost := math.Abs(float64(c.Steering) - desired)
		if cost < best {
			best = cost
			steer = i
		}
	}
	limit := trafficLimit
	if p.leavingPit {
		limit = 75
	}
	intent := vehicle.DriverIntent{Steering: float64(steer), Accelerate: speed < limit, Brake: speed > limit+7, DamageEnabled: f.DamageEnabled}
	yaw := int16(s.Body.Heading - track.Heading)
	if p.recoveringBrake {
		intent.Accelerate = false
		intent.Brake = true
		if float64(s.Body.Speed) < 4*(128*44.0/75) {
			p.recoveringBrake = false
		}
	} else {
		if p.reverseDirection == 0 && math.Abs(speed) < 30 && math.Abs(float64(yaw)) > 6000 {
			p.reverseDirection = -1
			if yaw < 0 {
				p.reverseDirection = 1
			}
		}
		if p.reverseDirection != 0 {
			intent.Accelerate = false
			if math.Abs(float64(yaw)) < 2000 || yaw < 0 && p.reverseDirection < 0 || yaw > 0 && p.reverseDirection > 0 {
				p.reverseDirection = 0
				p.recoveringBrake = true
				intent.Brake = true
			} else {
				intent.Brake = false
				intent.Reverse = true
				intent.Steering = float64(p.reverseDirection)
				steer = p.reverseDirection
			}
		}
	}
	return vehicle.DriverIntent{Steering: float64(steer), Accelerate: intent.Accelerate, Brake: intent.Brake, Reverse: intent.Reverse}
}

// Step forecasts copies of the original force and input-ramp state, then
// returns one ordinary steering/pedal intent for the actual game update.
func (p *Pilot) Step(f Frame) vehicle.DriverIntent {
	base := p.baseline(f)
	s := f.State
	if float64(s.Body.Speed) < 90*speedPerMPH || base.Reverse || p.recoveringBrake {
		return base
	}
	bestCost := math.Inf(1)
	best := base
	config := f.Config
	for _, cap := range []float64{170, 190, 210, 230} {
		for keyIndex, key := range []float64{base.Steering, 0, -1, 1} {
			if keyIndex > 0 && key == base.Steering {
				continue
			}
			v := vehicle.Vehicle{Config: config, State: s, World: p.world, Curve: p.curve, Surface: p.surface}
			d := vehicle.Driver{State: f.Driver}
			lateral := p.lateral
			cost, advance := 0.0, 0.0
			previous := s.Progress
			var first vehicle.DriverIntent
			for tick := 0; tick < p.horizon; tick++ {
				state := v.State
				track := p.world.TrackPlacement(state.Progress, state.Lateral<<4)
				forward, _ := p.world.VelocityOnTrack(worldscene.Vector2{X: state.Body.VelocityX, Y: state.Body.VelocityY}, track.Heading)
				speed := float64(forward) / speedPerMPH
				if tick > 0 {
					lateral += max(-100, min(100, p.laneTarget-lateral))
				}
				look := p.steerLook
				if look == 0 {
					look = 12
				}
				ahead := uint16((int(state.Progress) + int(max(speed, 40)*progressPerMPHTick*look)) % trackLength)
				target := p.world.TrackPlacement(ahead, int16(lateral))
				heading := p.world.Direction(target.X-int16(state.Body.PositionX>>16), target.Y-int16(state.Body.PositionY>>16))
				desired := max(-3200, min(3200, float64(int16(heading-state.Body.Heading))*.7-float64(state.Body.YawVelocity)*.5+float64(int16(state.Body.TravelHeading-state.Body.Heading))*.5))
				steer, steerCost := 0.0, math.Inf(1)
				for _, k := range []float64{0, -1, 1} {
					clone := d
					c := clone.Step(vehicle.DriverContext{Steering: state.Steering, Damping: state.SteeringDamping, Increment: state.SteeringIncrement, Speed: state.Body.Speed, Failures: state.Engine.Failures}, vehicle.DriverIntent{Steering: k})
					cst := math.Abs(float64(c.Steering) - desired)
					if cst < steerCost {
						steer, steerCost = k, cst
					}
				}
				if tick < 3 {
					steer = key
				}
				in := vehicle.DriverIntent{Steering: steer, Accelerate: speed < cap, Brake: speed > cap+5, DamageEnabled: f.DamageEnabled}
				if tick == 0 {
					first = in
				}
				c := d.Step(vehicle.DriverContext{Steering: state.Steering, Damping: state.SteeringDamping, Increment: state.SteeringIncrement, Speed: state.Body.Speed, Failures: state.Engine.Failures}, vehicle.DriverIntent{Steering: in.Steering, Accelerate: in.Accelerate, Brake: in.Brake, DamageEnabled: f.DamageEnabled})
				e := v.Step(c)
				step := int(v.State.Progress) - int(previous)
				if step > trackLength/2 {
					step -= trackLength
				}
				if step < -trackLength/2 {
					step += trackLength
				}
				advance += float64(step)
				previous = v.State.Progress
				if e.Impact != 0 || v.State.Engine.Failures != 0 {
					cost += 1000000
					break
				}
				inner, outer := p.world.TrackLimits(v.State.Progress)
				lat := float64(v.State.Lateral)
				distance := max(float64(inner+48)-lat, lat-float64(outer-48))
				if distance > 0 {
					cost += 1000 + distance*distance*10
				}
				error := lat - p.laneTarget/16
				cost += error * error * .001
				tangent := p.world.TrackPlacement(v.State.Progress, v.State.Lateral<<4).Heading
				course := max(0, math.Abs(float64(int16(v.State.Body.TravelHeading-tangent)))-600)
				slip := max(0, math.Abs(float64(int16(v.State.Body.TravelHeading-v.State.Body.Heading)))-1000)
				cost += course*course*.00003 + slip*slip*.00001
				if f.Traffic {
					for i := 1; i < len(f.Cars); i++ {
						other := f.Cars[i]
						if other.Next == i {
							continue
						}
						gap := float64(int(f.Cars[0].Distance) - int(other.Distance))
						if gap > 19800 {
							gap -= 39600
						}
						if gap < -19800 {
							gap += 39600
						}
						along := gap + float64(other.Speed)/256*float64(tick+1) - advance
						if math.Abs(along) > 120 {
							continue
						}
						otherLane := float64(other.Lateral) + max(-100, min(100, float64(other.LateralVelocity)))*float64(min(tick+1, 8))
						across := math.Abs(float64(v.State.Lateral)*16 - otherLane)
						if across < 1000 {
							cost += (1000 - across) * .4 * (1 - math.Abs(along)/120)
						}
					}
				}
			}
			cost -= advance
			if cost < bestCost {
				bestCost, best = cost, first
			}
		}
	}
	return best
}
