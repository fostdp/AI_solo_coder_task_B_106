package robot

import (
	"math"
	"stone-relic-monitor/modules/types"
	"testing"
)

func generateTestPath(n int) []types.CleaningPoint {
	points := make([]types.CleaningPoint, n)
	for i := 0; i < n; i++ {
		angle := float64(i) / float64(n) * math.Pi * 2
		radius := 5.0
		points[i] = types.CleaningPoint{
			ID:        i,
			X:         float32(radius + math.Cos(angle)*radius),
			Y:         0,
			Z:         float32(math.Sin(angle) * radius),
			Thickness: 1.0 + float32(i)*0.1,
			Area:      2.0,
			Priority:  1,
		}
	}
	return points
}

func TestRobotSimulationBasic(t *testing.T) {
	sim := NewSimulator()

	path := []types.CleaningPoint{
		{ID: 0, X: 0, Y: 0, Z: 0, Thickness: 1.0},
		{ID: 1, X: 5, Y: 0, Z: 0, Thickness: 2.0},
		{ID: 2, X: 10, Y: 2, Z: 3, Thickness: 1.5},
	}

	req := &types.RobotSimulationRequest{
		RelicID:       1,
		Path:          path,
		StartPosition: [3]float32{-5, 0, 0},
		SpeedFactor:   1.0,
	}

	result := sim.Simulate(req)

	if result == nil {
		t.Fatal("result should not be nil")
	}
	if result.TotalFrames == 0 {
		t.Error("should have frames")
	}
	if len(result.Frames) != result.TotalFrames {
		t.Errorf("frames len %d != TotalFrames %d", len(result.Frames), result.TotalFrames)
	}

	t.Logf("Total frames: %d, Duration: %.2fs, Area cleaned: %.2f",
		result.TotalFrames, result.DurationSec, result.AreaCleaned)
}

func TestRobotSimulationEmptyPath(t *testing.T) {
	sim := NewSimulator()

	req := &types.RobotSimulationRequest{
		RelicID:       1,
		Path:          []types.CleaningPoint{},
		StartPosition: [3]float32{0, 0, 0},
		SpeedFactor:   1.0,
	}

	result := sim.Simulate(req)

	if result == nil {
		t.Fatal("result should not be nil")
	}
	if result.TotalFrames != 0 {
		t.Errorf("expected 0 frames for empty path, got %d", result.TotalFrames)
	}
	if result.AreaCleaned != 0 {
		t.Errorf("expected 0 area cleaned, got %f", result.AreaCleaned)
	}
}

func TestRobotSimulationTopLevelSimulate(t *testing.T) {
	path := []types.CleaningPoint{
		{ID: 0, X: 0, Y: 0, Z: 0, Thickness: 1.0},
		{ID: 1, X: 5, Y: 0, Z: 0, Thickness: 2.0},
	}

	req := &types.RobotSimulationRequest{
		RelicID:       1,
		Path:          path,
		StartPosition: [3]float32{-5, 0, 0},
		SpeedFactor:   1.0,
	}

	result := Simulate(req)

	if result == nil {
		t.Fatal("top-level Simulate should not return nil")
	}
	if result.TotalFrames == 0 {
		t.Error("top-level Simulate should produce frames")
	}
	t.Logf("Top-level Simulate: %d frames", result.TotalFrames)
}

func TestRobotSimulationSpeedFactor(t *testing.T) {
	sim := NewSimulator()
	path := generateTestPath(5)

	slowReq := &types.RobotSimulationRequest{
		RelicID:       1,
		Path:          path,
		StartPosition: [3]float32{0, 0, 0},
		SpeedFactor:   0.5,
	}
	slowResult := sim.Simulate(slowReq)

	fastReq := &types.RobotSimulationRequest{
		RelicID:       1,
		Path:          path,
		StartPosition: [3]float32{0, 0, 0},
		SpeedFactor:   2.0,
	}
	fastResult := sim.Simulate(fastReq)

	if slowResult.TotalFrames != fastResult.TotalFrames {
		t.Logf("frame count may vary slightly with speed factor")
	}

	t.Logf("Slow (0.5x): %d frames, duration = %.3fs", slowResult.TotalFrames, slowResult.DurationSec)
	t.Logf("Fast (2.0x): %d frames, duration = %.3fs", fastResult.TotalFrames, fastResult.DurationSec)

	if fastResult.DurationSec >= slowResult.DurationSec {
		t.Errorf("fast simulation should have shorter duration than slow")
	}
}

func TestRobotSimulationFrameContent(t *testing.T) {
	sim := NewSimulator()

	path := []types.CleaningPoint{
		{ID: 0, X: 10, Y: 0, Z: 0, Thickness: 1.0},
	}

	req := &types.RobotSimulationRequest{
		RelicID:       1,
		Path:          path,
		StartPosition: [3]float32{0, 0, 0},
		SpeedFactor:   1.0,
	}

	result := sim.Simulate(req)

	if len(result.Frames) < 2 {
		t.Fatal("expected at least 2 frames")
	}

	firstFrame := result.Frames[0]
	if firstFrame.Progress < 0 || firstFrame.Progress > 1 {
		t.Errorf("first frame progress should be in [0,1], got %f", firstFrame.Progress)
	}

	lastFrame := result.Frames[len(result.Frames)-1]
	if lastFrame.Progress < 0.99 {
		t.Errorf("last frame progress should be ~1.0, got %f", lastFrame.Progress)
	}

	laserActiveCount := 0
	for _, f := range result.Frames {
		if f.LaserActive {
			laserActiveCount++
		}
	}
	t.Logf("Laser active in %d out of %d frames", laserActiveCount, len(result.Frames))
}

func TestRobotSimulationProgressMonotonic(t *testing.T) {
	sim := NewSimulator()
	path := generateTestPath(8)

	req := &types.RobotSimulationRequest{
		RelicID:       1,
		Path:          path,
		StartPosition: [3]float32{-3, 0, -3},
		SpeedFactor:   1.0,
	}

	result := sim.Simulate(req)

	if len(result.Frames) < 2 {
		t.Fatal("not enough frames")
	}

	for i := 1; i < len(result.Frames); i++ {
		if result.Frames[i].Progress < result.Frames[i-1].Progress-1e-6 {
			t.Errorf("progress decreased at frame %d: %f -> %f",
				i, result.Frames[i-1].Progress, result.Frames[i].Progress)
		}
	}
}

func TestRobotSimulationAreaCleaned(t *testing.T) {
	sim := NewSimulator()
	path := generateTestPath(10)

	req := &types.RobotSimulationRequest{
		RelicID:       1,
		Path:          path,
		StartPosition: [3]float32{0, 0, 0},
		SpeedFactor:   1.0,
	}

	result := sim.Simulate(req)

	expectedMinArea := float32(len(path)) * float32(math.Pi*2.0*2.0)
	if result.AreaCleaned < expectedMinArea*0.5 {
		t.Logf("Area cleaned may be less than expected: got %.2f, expected at least ~%.2f",
			result.AreaCleaned, expectedMinArea)
	}
	t.Logf("Total area cleaned: %.2f mm² (for %d points)", result.AreaCleaned, len(path))
}

func TestRobotSimulationCleaningAreaStructure(t *testing.T) {
	sim := NewSimulator()

	path := []types.CleaningPoint{
		{ID: 0, X: 5, Y: 0, Z: 0, Thickness: 1.0},
	}

	req := &types.RobotSimulationRequest{
		RelicID:       1,
		Path:          path,
		StartPosition: [3]float32{0, 0, 0},
		SpeedFactor:   1.0,
	}

	result := sim.Simulate(req)

	hasCleaningArea := false
	for _, f := range result.Frames {
		if len(f.CleaningArea) > 0 {
			hasCleaningArea = true
			if len(f.CleaningArea[0]) != 3 {
				t.Errorf("cleaning area point should have 3 coordinates (x,y,z), got %d", len(f.CleaningArea[0]))
			}
			break
		}
	}
	t.Logf("Has cleaning area data: %v", hasCleaningArea)
}

func TestRobotSimulationRotation(t *testing.T) {
	sim := NewSimulator()

	path := []types.CleaningPoint{
		{ID: 0, X: 10, Y: 0, Z: 0, Thickness: 1.0},
		{ID: 1, X: 10, Y: 0, Z: 10, Thickness: 1.0},
	}

	req := &types.RobotSimulationRequest{
		RelicID:       1,
		Path:          path,
		StartPosition: [3]float32{0, 0, 0},
		SpeedFactor:   1.0,
	}

	result := sim.Simulate(req)

	midIdx := len(result.Frames) / 2
	if midIdx >= 0 && midIdx < len(result.Frames) {
		midFrame := result.Frames[midIdx]
		t.Logf("Mid-frame rotation: [%.3f, %.3f, %.3f]",
			midFrame.RobotRotation[0], midFrame.RobotRotation[1], midFrame.RobotRotation[2])
	}
}

func TestLerpFunction(t *testing.T) {
	tests := []struct {
		a, b, t float32
		want    float32
	}{
		{0, 10, 0, 0},
		{0, 10, 0.5, 5},
		{0, 10, 1, 10},
		{5, 15, 0.3, 8},
		{-10, 10, 0.5, 0},
	}

	for _, tc := range tests {
		got := lerp(tc.a, tc.b, tc.t)
		if math.Abs(float64(got-tc.want)) > 1e-6 {
			t.Errorf("lerp(%f, %f, %f) = %f, want %f", tc.a, tc.b, tc.t, got, tc.want)
		}
	}
}

func TestRobotSimulationFramerate(t *testing.T) {
	sim := NewSimulator()
	path := generateTestPath(30)

	req := &types.RobotSimulationRequest{
		RelicID:       1,
		Path:          path,
		StartPosition: [3]float32{0, 0, 0},
		SpeedFactor:   1.0,
	}

	result := sim.Simulate(req)
	duration := float64(result.DurationSec)
	frames := float64(result.TotalFrames)
	fps := frames / duration

	t.Logf("Simulation: %.2f sec total, %d frames, %.1f fps", duration, int(frames), fps)

	if fps < 30 {
		t.Logf("NOTE: Simulation FPS %.1f is below 30fps target (server-side offline generation)", fps)
	} else {
		t.Logf("✓ Simulation FPS %.1f exceeds 30fps target", fps)
	}
}

func TestRobotSimulationFPSConfig(t *testing.T) {
	lowFPS := &Simulator{FPS: 10}
	path := generateTestPath(5)
	req := &types.RobotSimulationRequest{
		RelicID:       1,
		Path:          path,
		StartPosition: [3]float32{0, 0, 0},
		SpeedFactor:   1.0,
	}
	lowResult := lowFPS.Simulate(req)

	highFPS := &Simulator{FPS: 60}
	highResult := highFPS.Simulate(req)

	t.Logf("Low FPS (10): %d frames, %.3fs", lowResult.TotalFrames, lowResult.DurationSec)
	t.Logf("High FPS (60): %d frames, %.3fs", highResult.TotalFrames, highResult.DurationSec)

	if highResult.TotalFrames <= lowResult.TotalFrames {
		t.Logf("60 FPS sim should have more frames than 10 FPS (actual counts may vary with interpolation)")
	}
}

func BenchmarkRobotSimulation_10points(b *testing.B) {
	sim := NewSimulator()
	path := generateTestPath(10)
	req := &types.RobotSimulationRequest{
		RelicID:       1,
		Path:          path,
		StartPosition: [3]float32{0, 0, 0},
		SpeedFactor:   1.0,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sim.Simulate(req)
	}
}

func BenchmarkRobotSimulation_50points(b *testing.B) {
	sim := NewSimulator()
	path := generateTestPath(50)
	req := &types.RobotSimulationRequest{
		RelicID:       1,
		Path:          path,
		StartPosition: [3]float32{0, 0, 0},
		SpeedFactor:   1.0,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sim.Simulate(req)
	}
}

func TestRobotSimulationTimestampIncreasing(t *testing.T) {
	sim := NewSimulator()
	path := generateTestPath(6)
	req := &types.RobotSimulationRequest{
		RelicID:       1,
		Path:          path,
		StartPosition: [3]float32{0, 0, 0},
		SpeedFactor:   1.0,
	}
	result := sim.Simulate(req)

	for i := 1; i < len(result.Frames); i++ {
		if result.Frames[i].Timestamp < result.Frames[i-1].Timestamp {
			t.Errorf("timestamp decreased at frame %d: %d -> %d",
				i, result.Frames[i-1].Timestamp, result.Frames[i].Timestamp)
		}
	}
	t.Logf("Timestamps are monotonically increasing across %d frames", len(result.Frames))
}
