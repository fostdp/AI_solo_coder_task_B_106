package robot

import (
	"math"

	"stone-relic-monitor/modules/types"
)

type Simulator struct {
	FPS int
}

func NewSimulator() *Simulator {
	return &Simulator{FPS: 30}
}

func lerp(a, b, t float32) float32 {
	return a + (b-a)*t
}

func (s *Simulator) Simulate(req *types.RobotSimulationRequest) *types.RobotSimulationResult {
	if req == nil || len(req.Path) == 0 {
		return &types.RobotSimulationResult{
			RelicID:     req.RelicID,
			Frames:      []types.RobotFrame{},
			TotalFrames: 0,
			DurationSec: 0,
			AreaCleaned: 0,
		}
	}

	fps := s.FPS
	if fps <= 0 {
		fps = 30
	}

	path := req.Path
	n := len(path)
	totalSegments := n - 1
	if totalSegments < 0 {
		totalSegments = 0
	}

	baseSpeed := float32(50.0)
	if req.SpeedFactor > 0 {
		baseSpeed *= req.SpeedFactor
	}

	var frames []types.RobotFrame
	areaCleaned := float32(0)
	cleanedPoints := make(map[int]bool)
	frameIdx := int64(0)

	startPos := req.StartPosition
	if startPos[0] == 0 && startPos[1] == 0 && startPos[2] == 0 && n > 0 {
		startPos = [3]float32{path[0].X, path[0].Y, path[0].Z}
	}

	laserRadius := float32(2.5)
	if req.LaserParams.PredictedEnergyDensity > 0 {
		laserRadius = float32(math.Sqrt(float64(req.LaserParams.PredictedEnergyDensity))) * 1.5
	}
	if laserRadius < 0.5 {
		laserRadius = 0.5
	}

	if n == 1 {
		_ = startPos
		singlePt := path[0]
		nPts := 16
		cleaningArea := make([][]float32, nPts)
		for a := 0; a < nPts; a++ {
			angle := 2.0 * math.Pi * float64(a) / float64(nPts)
			cleaningArea[a] = []float32{
				singlePt.X + laserRadius*float32(math.Cos(angle)),
				singlePt.Y + laserRadius*0.2,
				singlePt.Z + laserRadius*float32(math.Sin(angle)),
			}
		}
		frames = append(frames, types.RobotFrame{
			Timestamp:      0,
			RobotPosition:  [3]float32{singlePt.X, singlePt.Y, singlePt.Z},
			RobotRotation:  [3]float32{0, 0, 0},
			CurrentPointID: singlePt.ID,
			LaserActive:    true,
			CleaningArea:   cleaningArea,
			Progress:       0.0,
		})
		frames = append(frames, types.RobotFrame{
			Timestamp:      int64(1000 / fps),
			RobotPosition:  [3]float32{singlePt.X, singlePt.Y, singlePt.Z},
			RobotRotation:  [3]float32{0, 0, 0},
			CurrentPointID: singlePt.ID,
			LaserActive:    false,
			CleaningArea:   cleaningArea,
			Progress:       1.0,
		})
		areaCleaned = float32(math.Pi) * laserRadius * laserRadius
		return &types.RobotSimulationResult{
			RelicID:     req.RelicID,
			Frames:      frames,
			TotalFrames: len(frames),
			DurationSec: float32(len(frames)) / float32(fps),
			AreaCleaned: areaCleaned,
		}
	}

	_ = startPos

	for seg := 0; seg < totalSegments; seg++ {
		from := path[seg]
		to := path[seg+1]

		segDist := float32(math.Sqrt(
			math.Pow(float64(to.X-from.X), 2) +
				math.Pow(float64(to.Y-from.Y), 2) +
				math.Pow(float64(to.Z-from.Z), 2)))

		stepsInSegment := 1
		if baseSpeed > 0 && segDist > 0 {
			timeSeconds := float64(segDist) / float64(baseSpeed)
			stepsInSegment = int(math.Max(1, math.Ceil(timeSeconds*float64(fps))))
		}

		for step := 0; step < stepsInSegment; step++ {
			t := float32(step) / float32(stepsInSegment)

			newPos := [3]float32{
				lerp(from.X, to.X, t),
				lerp(from.Y, to.Y, t),
				lerp(from.Z, to.Z, t),
			}

			rotation := [3]float32{0, 0, 0}
			if segDist > 0.001 {
				dx := float64(to.X - from.X)
				dz := float64(to.Z - from.Z)
				rotation[1] = float32(math.Atan2(dz, dx))
				if math.Abs(float64(to.Y-from.Y)) > 0.001 {
					dy := float64(to.Y - from.Y)
					horiz := math.Sqrt(dx*dx + dz*dz)
					rotation[0] = float32(math.Atan2(dy, horiz))
				}
			}

			var cleaningArea [][]float32
			laserActive := req.LaserParams.OptimalPower > 0 && req.LaserParams.OptimalSpeed > 0

			distToTarget := float32(math.Sqrt(
				math.Pow(float64(newPos[0]-to.X), 2) +
					math.Pow(float64(newPos[1]-to.Y), 2) +
					math.Pow(float64(newPos[2]-to.Z), 2)))

			if laserActive && distToTarget < laserRadius*3 {
				laserActive = true
				nPts := 16
				cleaningArea = make([][]float32, nPts)
				for a := 0; a < nPts; a++ {
					angle := 2.0 * math.Pi * float64(a) / float64(nPts)
					cleaningArea[a] = []float32{
						newPos[0] + laserRadius*float32(math.Cos(angle)),
						newPos[1] + laserRadius*0.2,
						newPos[2] + laserRadius*float32(math.Sin(angle)),
					}
				}
				if !cleanedPoints[seg+1] && distToTarget < laserRadius {
					cleanedPoints[seg+1] = true
					areaCleaned += to.Area
				}
			} else if distToTarget > laserRadius*3 {
				laserActive = false
			}

			progress := float32(seg) / float32(totalSegments)
			if totalSegments > 1 {
				progress += t / float32(totalSegments)
			}

			currentID := seg
			if t >= 0.5 && seg < n-1 {
				currentID = seg + 1
			}
			if currentID >= n {
				currentID = n - 1
			}

			frames = append(frames, types.RobotFrame{
				Timestamp:      frameIdx * int64(1000/fps),
				RobotPosition:  newPos,
				RobotRotation:  rotation,
				CurrentPointID: path[currentID].ID,
				LaserActive:    laserActive,
				CleaningArea:   cleaningArea,
				Progress:       progress,
			})
			frameIdx++
		}
	}

	if len(frames) > 0 {
		frames[len(frames)-1].Progress = 1.0
		frames[len(frames)-1].LaserActive = false
		if len(frames[len(frames)-1].CleaningArea) == 0 {
			nPts := 16
			cleaningArea := make([][]float32, nPts)
			lastP := path[n-1]
			for a := 0; a < nPts; a++ {
				angle := 2.0 * math.Pi * float64(a) / float64(nPts)
				cleaningArea[a] = []float32{
					lastP.X + laserRadius*float32(math.Cos(angle)),
					lastP.Y + laserRadius*0.2,
					lastP.Z + laserRadius*float32(math.Sin(angle)),
				}
			}
			frames[len(frames)-1].CleaningArea = cleaningArea
		}
	}

	totalFrames := len(frames)
	durationSec := float32(0)
	if fps > 0 {
		durationSec = float32(totalFrames) / float32(fps)
	}

	return &types.RobotSimulationResult{
		RelicID:     req.RelicID,
		Frames:      frames,
		TotalFrames: totalFrames,
		DurationSec: durationSec,
		AreaCleaned: areaCleaned,
	}
}

func Simulate(req *types.RobotSimulationRequest) *types.RobotSimulationResult {
	return NewSimulator().Simulate(req)
}
