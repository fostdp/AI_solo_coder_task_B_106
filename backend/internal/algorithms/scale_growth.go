package algorithms

import (
	"math"
	"stone-relic-monitor/internal/models"
)

const (
	SULFATE_GROWTH_RATE_BASE = 0.00001
	SO2_REACTION_EXPONENT    = 0.7
	HUMIDITY_CRITICAL        = 60.0
	TEMP_ACTIVATION_ENERGY   = 4000.0
)

func PredictScaleGrowth(p *models.ScaleGrowthPrediction) {
	p.PredictedThickness = make([]float32, p.Hours+1)
	p.PredictedThickness[0] = p.InitialThickness

	so2Factor := math.Pow(float64(p.SO2Concentration*0.001), SO2_REACTION_EXPONENT)
	humidityFactor := 1.0
	if float64(p.Humidity) > HUMIDITY_CRITICAL {
		humidityFactor = 1.0 + 2.5*math.Pow((float64(p.Humidity)-HUMIDITY_CRITICAL)/(100.0-HUMIDITY_CRITICAL), 2)
	} else {
		humidityFactor = 0.3 + 0.7*math.Pow(float64(p.Humidity)/HUMIDITY_CRITICAL, 3)
	}
	tempFactor := math.Exp(TEMP_ACTIVATION_ENERGY_RATIO(float64(p.Temperature)))

	growthRate := SULFATE_GROWTH_RATE_BASE * so2Factor * humidityFactor * tempFactor
	p.GrowthRate = float32(growthRate)

	for h := 1; h <= p.Hours; h++ {
		hourlyGrowth := growthRate * (1.0 + 0.1*math.Sin(2*math.Pi*float64(h)/24))
		saturationFactor := 1.0 - math.Exp(-float64(p.PredictedThickness[h-1])/5.0)
		p.PredictedThickness[h] = p.PredictedThickness[h-1] + float32(hourlyGrowth*(1.0-saturationFactor))

		if p.PredictedThickness[h] > 10.0 {
			p.PredictedThickness[h] = 10.0
		}
	}

	p.FinalThickness = p.PredictedThickness[p.Hours]
	p.SaturationFactor = float32(1.0 - math.Exp(-float64(p.FinalThickness)/5.0))
}

func TEMP_ACTIVATION_ENERGY_RATIO(tempC float64) float64 {
	const t0 = 293.15
	tempK := tempC + 273.15
	return TEMP_ACTIVATION_ENERGY / 8.314 * (1.0/t0 - 1.0/tempK)
}

func CalculateDailyGrowthRate(thicknessHistory []float32, timestampsHours []int) float32 {
	if len(thicknessHistory) < 2 {
		return 0
	}
	deltaT := float64(timestampsHours[len(timestampsHours)-1] - timestampsHours[0])
	if deltaT <= 0 {
		return 0
	}
	deltaH := float64(thicknessHistory[len(thicknessHistory)-1] - thicknessHistory[0])
	return float32(deltaH / deltaT * 24)
}
