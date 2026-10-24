package sound

// Sample identifies one recovered waveform, including the short failure sound
// that shares part of the collision recording.
type Sample uint8

const (
	EngineSample Sample = iota
	NoiseSample
	CollisionSample
	FailureSample
)

// Voice is a normal PCM playback request. Rate is samples per second, Level
// uses the original 0..64 gain range, and Restart changes only on a new cue.
type Voice struct {
	Sample  Sample
	Rate    float64
	Level   uint8
	Restart uint64
}

// Controller retains the game's pitch interpolation, noise envelope and effect
// priorities. It does not execute processor code or emulate an audio device.
type Controller struct {
	Pitch                           [3]uint16
	PitchStep                       [3]int16
	EngineLevel                     [3]uint8
	EngineNumerator, NoiseNumerator uint32
	NoisePitch, NoiseDrift          int16
	NoiseAccumulator                uint16
	NoiseLevel                      uint8
	NoiseStep                       int8
	RandomCountdown                 uint8
	Requests                        [4]uint8
	EffectTicks, Priority           uint8
	Voices                          [4]Voice
}

const originalPlaybackClock = 3546895.0

// NewController starts the original engine and continuous road-noise voices.
func NewController() Controller {
	c := Controller{Pitch: [3]uint16{2048, 2048, 2048}, EngineNumerator: 0x2eea4d,
		NoiseNumerator: (3560000 / 10000) * 8192, NoisePitch: 8192, RandomCountdown: 30}
	for i := range 3 {
		c.Voices[i] = Voice{Sample: EngineSample, Rate: originalPlaybackClock / 323}
	}
	c.Voices[3] = Voice{Sample: NoiseSample, Rate: originalPlaybackClock / 356}
	return c
}

func playbackRate(numerator uint32, divisor uint16) float64 {
	period := uint16(numerator / uint32(divisor))
	if period == 0 {
		return originalPlaybackClock / 65536
	}
	return originalPlaybackClock / float64(period)
}

// Step follows one original audio-control update. Random shares the game's
// existing generator, because the noise variation also advances its sequence.
func (c *Controller) Step(random func() uint16) {
	for i := range c.Pitch {
		c.Pitch[i] += uint16(c.PitchStep[i])
	}
	level := int16(c.NoiseAccumulator + uint16(int16(c.NoiseStep)))
	level = max(int16(0), min(int16(508), level))
	c.NoiseAccumulator = uint16(level)
	c.NoiseLevel = uint8(level >> 2)
	if c.EffectTicks == 0 {
		c.Priority = uint8(max(int8(c.NoiseLevel)-40, 0))
	}
	c.RandomCountdown--
	if c.RandomCountdown == 0 {
		c.RandomCountdown = 30
		c.NoiseDrift = max(int16(-1024), min(int16(1024), c.NoiseDrift+int16(int8(random()))))
	}
	for i := range c.Pitch {
		pitch := c.Pitch[i]
		if int16(pitch) < 0 {
			pitch = 32767
		}
		pitch = max(uint16(2048), pitch)
		c.Voices[i].Rate = playbackRate(c.EngineNumerator, pitch)
		c.Voices[i].Level = c.EngineLevel[i] >> 1
	}
	for i, request := range c.Requests {
		if request == 0 {
			continue
		}
		c.Requests[i] = 0
		if request < c.Priority {
			continue
		}
		c.Priority = request
		effect := &c.Voices[3]
		effect.Restart++
		effect.Sample = FailureSample
		effect.Level = request >> 1
		periods := [4]uint16{0, 800, 680, 490}
		durations := [4]uint8{27, 22, 20, 14}
		period := periods[i]
		if i == 0 {
			effect.Sample = CollisionSample
			period = 570 + (uint16(request&7) << 5)
		}
		effect.Rate = originalPlaybackClock / float64(period)
		c.EffectTicks = durations[i]
		return
	}
	if c.EffectTicks != 0 {
		c.EffectTicks--
		if c.EffectTicks != 0 {
			return
		}
		effect := &c.Voices[3]
		effect.Sample = NoiseSample
		effect.Level = 0
		effect.Restart++
		c.NoiseAccumulator = 0
		c.NoiseStep = int8(c.NoiseLevel)
		c.Priority = uint8(max(int8(c.NoiseStep)-40, 0))
		c.NoiseLevel = 0
	}
	pitch := c.NoiseDrift + c.NoisePitch
	pitch = max(int16(7168), min(int16(9216), pitch))
	c.Voices[3].Rate = playbackRate(c.NoiseNumerator, uint16(pitch))
	c.Voices[3].Level = c.NoiseLevel >> 1
}

// Reset starts a new driving session and restarts each recovered waveform.
func (c *Controller) Reset() {
	next := NewController()
	for i := range next.Voices {
		next.Voices[i].Restart = c.Voices[i].Restart + 1
	}
	*c = next
}

// ResetDrivingControls follows the original pause and resume reset. Engine
// waveforms keep running at their previous rates while muted; only the noise
// waveform restarts. The next audio update installs the reset pitch periods.
func (c *Controller) ResetDrivingControls() {
	next := NewController()
	next.EngineNumerator, next.NoiseNumerator = c.EngineNumerator, c.NoiseNumerator
	next.Voices = c.Voices
	for i := range next.Voices {
		next.Voices[i].Level = 0
	}
	next.Voices[3].Sample = NoiseSample
	next.Voices[3].Restart++
	*c = next
}
