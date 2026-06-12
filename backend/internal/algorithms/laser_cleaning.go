package algorithms

import (
	"math"
	"stone-relic-monitor/internal/config"
	"stone-relic-monitor/internal/models"
)

const (
	DEFAULT_ABLATION_THRESHOLD = 1.2
	SPOT_DIAMETER              = 0.1
	THERMAL_DIFFUSIVITY        = 0.001
	MOLAR_MASS_CASO4           = 136.14
	ENTHALPY_VAPORIZATION      = 1.8e6
	DENSITY_SCALE              = 2.32e3
	SAFETY_MARGIN              = 0.85
)

var defaultLaserCfg = config.LaserConfig{
	DefaultPower:     200.0,
	MinPower:         50.0,
	MaxPower:         300.0,
	MinPulseDuration: 200.0,
	MaxPulseDuration: 2000.0,
	MinScanSpeed:     10.0,
	MaxScanSpeed:     200.0,
	PowerStep:        10.0,
	PulseStep:        100.0,
	SpeedStep:        5.0,
}

var laserCfg = defaultLaserCfg

func SetLaserConfig(cfg config.LaserConfig) {
	if cfg.DefaultPower > 0 {
		laserCfg.DefaultPower = cfg.DefaultPower
	}
	if cfg.MinPower > 0 {
		laserCfg.MinPower = cfg.MinPower
	}
	if cfg.MaxPower > 0 {
		laserCfg.MaxPower = cfg.MaxPower
	}
	if cfg.MinPulseDuration > 0 {
		laserCfg.MinPulseDuration = cfg.MinPulseDuration
	}
	if cfg.MaxPulseDuration > 0 {
		laserCfg.MaxPulseDuration = cfg.MaxPulseDuration
	}
	if cfg.MinScanSpeed > 0 {
		laserCfg.MinScanSpeed = cfg.MinScanSpeed
	}
	if cfg.MaxScanSpeed > 0 {
		laserCfg.MaxScanSpeed = cfg.MaxScanSpeed
	}
	if cfg.PowerStep > 0 {
		laserCfg.PowerStep = cfg.PowerStep
	}
	if cfg.PulseStep > 0 {
		laserCfg.PulseStep = cfg.PulseStep
	}
	if cfg.SpeedStep > 0 {
		laserCfg.SpeedStep = cfg.SpeedStep
	}
}

var materialAblationParams = map[string]struct {
	threshold  float64
	efficiency float64
}{
	"gypsum":         {1.2, 0.72},
	"calcium_sulfate":{1.2, 0.72},
	"calcite":        {2.8, 0.85},
	"dolomite":       {2.5, 0.80},
	"silicate":       {3.5, 0.90},
	"default":        {2.0, 0.78},
}

func safeDiv(a, b float64) float64 {
	if math.Abs(b) < 1e-15 {
		return 0
	}
	return a / b
}

func isFinite(f float64) bool {
	return !math.IsNaN(f) && !math.IsInf(f, 0)
}

func PredictLaserCleaning(req *models.LaserCleaningRequest) *models.LaserCleaningResult {
	params, ok := materialAblationParams[req.MaterialType]
	if !ok {
		params = materialAblationParams["default"]
	}

	result := &models.LaserCleaningResult{
		AblationThreshold: float32(params.threshold),
		Confidence:        0.88,
	}

	targetDepth := float64(req.TargetThickness)
	if targetDepth <= 0 {
		result.SafetyWarning = "错误：目标厚度必须为正值"
		result.Confidence = 0
		return result
	}

	if params.efficiency <= 0 {
		result.SafetyWarning = "错误：材料效率参数异常"
		result.Confidence = 0
		return result
	}

	spotArea := math.Pi * math.Pow(SPOT_DIAMETER/2, 2)
	if spotArea <= 0 {
		result.SafetyWarning = "错误：光斑面积计算异常"
		result.Confidence = 0
		return result
	}

	minEnergyDensity := safeDiv(params.threshold, params.efficiency)
	targetEnergyDensity := safeDiv(DENSITY_SCALE*ENTHALPY_VAPORIZATION*targetDepth/MOLAR_MASS_CASO4*1e6, params.efficiency)
	optimalEnergyDensity := (minEnergyDensity + targetEnergyDensity) / 2 * SAFETY_MARGIN

	if !isFinite(optimalEnergyDensity) {
		result.SafetyWarning = "错误：最优能量密度计算异常"
		result.Confidence = 0
		return result
	}

	pulseDuration := laserCfg.MinPulseDuration + (laserCfg.MaxPulseDuration-laserCfg.MinPulseDuration)/2
	scanSpeed := laserCfg.MinScanSpeed + (laserCfg.MaxScanSpeed-laserCfg.MinScanSpeed)/2
	laserPower := laserCfg.DefaultPower

	minError := math.MaxFloat64
	for p := laserCfg.MinPower; p <= laserCfg.MaxPower; p += laserCfg.PowerStep {
		for pd := laserCfg.MinPulseDuration; pd <= laserCfg.MaxPulseDuration; pd += laserCfg.PulseStep {
			for ss := laserCfg.MinScanSpeed; ss <= laserCfg.MaxScanSpeed; ss += laserCfg.SpeedStep {
				pulseEnergy := p * pd / 1e6
				energyDensity := safeDiv(pulseEnergy, spotArea)
				overlap := 1 - safeDiv(ss*pd/1e6, SPOT_DIAMETER*0.8)
				if overlap < 0.1 || overlap > 0.9 {
					continue
				}
				effectiveEnergy := energyDensity * (1 + overlap*0.5)

				heatPenetration := math.Sqrt(4 * THERMAL_DIFFUSIVITY * pd / 1e6)
				if heatPenetration < targetDepth*0.8 || heatPenetration > targetDepth*3.0 {
					continue
				}

				edError := math.Abs(effectiveEnergy - optimalEnergyDensity)
				thresholdRatio := safeDiv(effectiveEnergy, params.threshold)
				if thresholdRatio < 1.05 || thresholdRatio > 3.0 {
					continue
				}

				totalError := edError*2.0 + math.Abs(thresholdRatio-1.8)*10.0
				if totalError < minError {
					minError = totalError
					laserPower = p
					pulseDuration = pd
					scanSpeed = ss
					result.PredictedEnergyDensity = float32(energyDensity)
				}
			}
		}
	}

	result.OptimalPower = float32(math.Min(laserCfg.MaxPower, math.Max(laserCfg.MinPower, laserPower)))
	result.OptimalPulse = float32(math.Min(laserCfg.MaxPulseDuration, math.Max(laserCfg.MinPulseDuration, pulseDuration)))
	result.OptimalSpeed = float32(math.Min(laserCfg.MaxScanSpeed, math.Max(laserCfg.MinScanSpeed, scanSpeed)))

	pulseEnergy := float64(result.OptimalPower) * float64(result.OptimalPulse) / 1e6
	energyDensity := safeDiv(pulseEnergy, spotArea)
	overlap := 1 - safeDiv(float64(result.OptimalSpeed)*float64(result.OptimalPulse)/1e6, SPOT_DIAMETER*0.8)
	effectiveEnergy := energyDensity * (1 + overlap*0.5)

	ablationDepth := safeDiv(
		(effectiveEnergy-params.threshold)*params.efficiency*MOLAR_MASS_CASO4,
		DENSITY_SCALE*ENTHALPY_VAPORIZATION,
	) * 1e6
	if !isFinite(ablationDepth) {
		ablationDepth = 0
	}
	result.PredictedDepth = float32(
		math.Min(targetDepth*1.05,
			math.Max(0, ablationDepth)))

	thresholdRatio := safeDiv(effectiveEnergy, params.threshold)
	if thresholdRatio > 2.5 {
		result.SafetyWarning = "警告：能量密度接近石材基体损伤阈值，建议先进行小区域试验"
		result.Confidence *= 0.85
	} else if thresholdRatio < 1.1 {
		result.SafetyWarning = "注意：能量接近结垢烧蚀阈值，可能需要多遍扫描"
		result.Confidence *= 0.9
	} else {
		result.SafetyWarning = "参数安全，建议从0.8倍功率开始校准"
	}

	return result
}

func CalculateAblationDepth(laserPower float32, pulseDuration float32, energyDensity float32) float32 {
	params := materialAblationParams["calcium_sulfate"]
	spotArea := math.Pi * math.Pow(SPOT_DIAMETER/2, 2)
	if spotArea <= 0 {
		return 0
	}
	pulseEnergy := float64(laserPower) * float64(pulseDuration) / 1e6
	ed := safeDiv(pulseEnergy, spotArea)

	if ed < params.threshold {
		return 0
	}
	depth := safeDiv(
		(ed-params.threshold)*params.efficiency*MOLAR_MASS_CASO4,
		DENSITY_SCALE*ENTHALPY_VAPORIZATION,
	) * 1e6
	if !isFinite(depth) {
		return 0
	}
	return float32(depth)
}

func OptimizeCleaningParametersBatch(targets []models.LaserCleaningRequest) []models.LaserCleaningResult {
	results := make([]models.LaserCleaningResult, len(targets))
	for i, req := range targets {
		results[i] = *PredictLaserCleaning(&req)
	}
	return results
}
