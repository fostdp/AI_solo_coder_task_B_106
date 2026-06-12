package scaling

import (
	"math"
)

type WindDirection struct {
	Azimuth  float64
	Speed    float64
	Stability float64
}

type SurfaceOrientation struct {
	NormalAzimuth  float64
	NormalInclination float64
	SurfaceRoughness float64
	AreaFactor       float64
}

type OrientationGrowthModifier struct {
	Wind        WindDirection
	Surface     SurfaceOrientation
	CFDCalibrated bool
	CalibrationK   float64
}

func NewOrientationGrowthModifier(wind WindDirection, surface SurfaceOrientation) *OrientationGrowthModifier {
	return &OrientationGrowthModifier{
		Wind:           wind,
		Surface:        surface,
		CFDCalibrated:  false,
		CalibrationK:   1.0,
	}
}

func (m *OrientationGrowthModifier) CalculateWindIncidenceAngle() float64 {
	windAzimuth := m.Wind.Azimuth
	surfaceAzimuth := m.Surface.NormalAzimuth
	deltaAz := math.Abs(windAzimuth - surfaceAzimuth)
	if deltaAz > 180 {
		deltaAz = 360 - deltaAz
	}
	return deltaAz * math.Pi / 180
}

func (m *OrientationGrowthModifier) WindwardFactor() float64 {
	theta := m.CalculateWindIncidenceAngle()
	cosTheta := math.Cos(theta)
	if cosTheta < 0 {
		return 0.2 + 0.1*math.Abs(cosTheta)
	}
	cosIncline := math.Cos(m.Surface.NormalInclination * math.Pi / 180)
	windEffect := math.Pow(cosTheta, 0.8) * cosIncline
	baseFactor := 0.3 + 0.7*windEffect
	speedEnhance := 1.0 + 0.35*math.Log10(m.Wind.Speed+1)
	return baseFactor * speedEnhance * m.CalibrationK
}

func (m *OrientationGrowthModifier) DepositionVelocity() float64 {
	uStar := m.Wind.Speed * 0.08
	sc := 1.5
	sc23 := math.Pow(sc, -2.0/3.0)
	roughnessFactor := math.Pow(m.Surface.SurfaceRoughness/10.0, 0.25)
	vd := uStar * sc23 * (1 + roughnessFactor*0.5)
	if vd < 0.001 {
		vd = 0.001
	}
	return vd
}

func (m *OrientationGrowthModifier) GravitySettlingFactor() float64 {
	inclination := m.Surface.NormalInclination * math.Pi / 180
	gravityFactor := math.Sin(inclination)
	if gravityFactor < 0 {
		gravityFactor = 0
	}
	return 0.8 + 0.2*gravityFactor
}

func (m *OrientationGrowthModifier) TotalFactor() float64 {
	windFactor := m.WindwardFactor()
	depositionFactor := m.DepositionVelocity()
	gravityFactor := m.GravitySettlingFactor()
	combined := windFactor * (0.6 + 0.3*depositionFactor/0.01) * gravityFactor
	return math.Max(0.15, math.Min(2.5, combined))
}

type CFDWindField struct {
	GridSizeX int
	GridSizeY int
	GridSizeZ int
	UField    []float64
	VField    []float64
	WField    []float64
	TurbKinEnergy []float64
	Dissipation  []float64
}

func NewCFDWindField(nx, ny, nz int) *CFDWindField {
	total := nx * ny * nz
	return &CFDWindField{
		GridSizeX:      nx,
		GridSizeY:      ny,
		GridSizeZ:      nz,
		UField:         make([]float64, total),
		VField:         make([]float64, total),
		WField:         make([]float64, total),
		TurbKinEnergy:  make([]float64, total),
		Dissipation:    make([]float64, total),
	}
}

func (f *CFDWindField) idx(i, j, k int) int {
	return k*f.GridSizeX*f.GridSizeY + j*f.GridSizeX + i
}

func (f *CFDWindField) SampleWind(x, y, z float64) WindDirection {
	fx := math.Max(0, math.Min(float64(f.GridSizeX-1), x))
	fy := math.Max(0, math.Min(float64(f.GridSizeY-1), y))
	fz := math.Max(0, math.Min(float64(f.GridSizeZ-1), z))

	i0 := int(math.Floor(fx))
	j0 := int(math.Floor(fy))
	k0 := int(math.Floor(fz))
	i1 := i0 + 1
	j1 := j0 + 1
	k1 := k0 + 1
	if i1 >= f.GridSizeX { i1 = i0 }
	if j1 >= f.GridSizeY { j1 = j0 }
	if k1 >= f.GridSizeZ { k1 = k0 }

	fx1 := fx - float64(i0)
	fy1 := fy - float64(j0)
	fz1 := fz - float64(k0)

	trilerp := func(field []float64) float64 {
		c000 := field[f.idx(i0, j0, k0)]
		c100 := field[f.idx(i1, j0, k0)]
		c010 := field[f.idx(i0, j1, k0)]
		c110 := field[f.idx(i1, j1, k0)]
		c001 := field[f.idx(i0, j0, k1)]
		c101 := field[f.idx(i1, j0, k1)]
		c011 := field[f.idx(i0, j1, k1)]
		c111 := field[f.idx(i1, j1, k1)]

		c00 := c000*(1-fx1) + c100*fx1
		c10 := c010*(1-fx1) + c110*fx1
		c01 := c001*(1-fx1) + c101*fx1
		c11 := c011*(1-fx1) + c111*fx1

		c0 := c00*(1-fy1) + c10*fy1
		c1 := c01*(1-fy1) + c11*fy1

		return c0*(1-fz1) + c1*fz1
	}

	u := trilerp(f.UField)
	v := trilerp(f.VField)
	w := trilerp(f.WField)

	speed := math.Sqrt(u*u + v*v + w*w)
	azimuth := math.Atan2(u, v) * 180 / math.Pi
	if azimuth < 0 { azimuth += 360 }

	tke := trilerp(f.TurbKinEnergy)
	stability := math.Exp(-tke / 2.0)

	return WindDirection{
		Azimuth:   azimuth,
		Speed:     speed,
		Stability: stability,
	}
}

func (f *CFDWindField) CalibrateWithFieldData(sensorData []struct {
	X, Y, Z float64
	GrowthRate float64
	SO2        float64
	Humidity   float64
	BaseRate   float64
}) float64 {
	if len(sensorData) == 0 {
		return 1.0
	}

	var sumRatio float64
	var count int
	for _, d := range sensorData {
		wind := f.SampleWind(d.X, d.Y, d.Z)
		surface := SurfaceOrientation{
			NormalAzimuth:     180,
			NormalInclination: 20,
			SurfaceRoughness:  5.0,
			AreaFactor:        1.0,
		}
		modifier := NewOrientationGrowthModifier(wind, surface)
		modifier.CalibrationK = 1.0
		factor := modifier.TotalFactor()
		expected := d.BaseRate * factor
		if expected > 0 {
			sumRatio += d.GrowthRate / expected
			count++
		}
	}

	if count > 0 {
		return sumRatio / float64(count)
	}
	return 1.0
}

type ScaleGrowthWithOrientation struct {
	CFDField     *CFDWindField
	SO2Factor    float64
	HumidityFactor float64
	BaseRate     float64
	Orientations []SurfaceOrientation
}

func NewScaleGrowthWithOrientation(cfdField *CFDWindField) *ScaleGrowthWithOrientation {
	return &ScaleGrowthWithOrientation{
		CFDField: cfdField,
		SO2Factor: 0.7,
		HumidityFactor: 0.0,
		BaseRate:  0.00001,
	}
}

func (m *ScaleGrowthWithOrientation) PredictPointGrowth(
	x, y, z float64,
	initialThickness float64,
	so2Concentration float64,
	humidity float64,
	temperature float64,
	hours int,
	surface SurfaceOrientation,
) []float64 {
	predicted := make([]float64, hours+1)
	predicted[0] = initialThickness

	wind := m.CFDField.SampleWind(x, y, z)
	modifier := NewOrientationGrowthModifier(wind, surface)
	orientFactor := modifier.TotalFactor()

	so2Factor := math.Pow(so2Concentration*0.001, m.SO2Factor)

	humidFactor := 1.0
	if humidity > 60 {
		humidFactor = 1.0 + 2.5*math.Pow((humidity-60)/40, 2)
	} else {
		humidFactor = 0.3 + 0.7*math.Pow(humidity/60, 3)
	}

	tempRatio := 1.0/293.15 - 1.0/(temperature+273.15)
	tempFactor := math.Exp(4000 / 8.314 * tempRatio)

	growthRate := m.BaseRate * so2Factor * humidFactor * tempFactor * orientFactor

	for h := 1; h <= hours; h++ {
		diurnal := 1.0 + 0.1*math.Sin(2*math.Pi*float64(h)/24)
		hourlyGrowth := growthRate * diurnal
		saturation := 1.0 - math.Exp(-predicted[h-1]/5.0)
		predicted[h] = predicted[h-1] + float64(hourlyGrowth)*(1.0-saturation)

		if predicted[h] > 10.0 {
			predicted[h] = 10.0
		}
	}

	return predicted
}

func (m *ScaleGrowthWithOrientation) PredictFieldGrowth(
	points []struct {
		X, Y, Z float64
		Thickness float64
		Surface SurfaceOrientation
	},
	so2Concentration float64,
	humidity float64,
	temperature float64,
	hours int,
) [][]float64 {
	results := make([][]float64, len(points))
	for i, p := range points {
		results[i] = m.PredictPointGrowth(
			p.X, p.Y, p.Z,
			p.Thickness,
			so2Concentration, humidity, temperature,
			hours, p.Surface,
		)
	}
	return results
}

func GenerateSyntheticBuddhaCFDField() *CFDWindField {
	nx, ny, nz := 32, 32, 48
	field := NewCFDWindField(nx, ny, nz)

	windDir := 270.0 * math.Pi / 180
	baseSpeed := 3.5

	for k := 0; k < nz; k++ {
		heightRatio := float64(k) / float64(nz)
		speedProfile := baseSpeed * math.Pow(heightRatio+0.1, 0.18)

		for j := 0; j < ny; j++ {
			for i := 0; i < nx; i++ {
				fx := (float64(i)/float64(nx-1) - 0.5) * 2
				fy := (float64(j)/float64(ny-1) - 0.5) * 2
				fz := heightRatio

				distFromCenter := math.Sqrt(fx*fx + fy*fy)
				obstacleFactor := 1.0
				if fz < 0.7 && distFromCenter < 0.5 {
					obstacleFactor = distFromCenter * 2
				}

				uwake := 0.0
				if fx > 0 && fz < 0.6 {
					wakeDecay := math.Exp(-fx * 2.0)
					uwake = -speedProfile * 0.4 * wakeDecay
				}

				vortY := 0.0
				if distFromCenter < 0.6 && fz > 0.3 && fz < 0.8 {
					vortMag := speedProfile * 0.25
					vortY = -vortMag * fx * 2
				}

				u := speedProfile * obstacleFactor * math.Cos(windDir) + uwake
				v := speedProfile * obstacleFactor * math.Sin(windDir) + vortY
				w := -speedProfile * 0.05 * (1 - fz) * distFromCenter

				idx := field.idx(i, j, k)
				field.UField[idx] = u
				field.VField[idx] = v
				field.WField[idx] = w

				tke := 0.5 * (u*u + v*v + w*w) * 0.08
				if fz < 0.2 {
					tke *= 2.5
				}
				if distFromCenter < 0.3 && fz > 0.3 && fz < 0.7 {
					tke *= 1.8
				}
				field.TurbKinEnergy[idx] = tke
				field.Dissipation[idx] = tke * 0.5 / math.Max(0.1, fz*10)
			}
		}
	}

	return field
}

type OrientationGrowthResult struct {
	PointIndex      int
	X, Y, Z         float64
	InitialThickness float64
	FinalThickness   float64
	TotalGrowth      float64
	OrientFactor     float64
	WindAzimuth      float64
	WindSpeed        float64
	WindwardScore    float64
}

func CalculateOrientedGrowthComparison(
	field *CFDWindField,
	points []struct {
		X, Y, Z float64
		Thickness float64
		Surface SurfaceOrientation
	},
	so2, humidity, temp float64,
	hours int,
) []OrientationGrowthResult {
	model := NewScaleGrowthWithOrientation(field)
	results := make([]OrientationGrowthResult, len(points))

	for i, p := range points {
		wind := field.SampleWind(p.X, p.Y, p.Z)
		modifier := NewOrientationGrowthModifier(wind, p.Surface)
		prediction := model.PredictPointGrowth(
			p.X, p.Y, p.Z, p.Thickness, so2, humidity, temp, hours, p.Surface,
		)
		results[i] = OrientationGrowthResult{
			PointIndex:       i,
			X:                p.X,
			Y:                p.Y,
			Z:                p.Z,
			InitialThickness: p.Thickness,
			FinalThickness:   prediction[len(prediction)-1],
			TotalGrowth:      prediction[len(prediction)-1] - p.Thickness,
			OrientFactor:     modifier.TotalFactor(),
			WindAzimuth:      wind.Azimuth,
			WindSpeed:        wind.Speed,
			WindwardScore:    modifier.WindwardFactor(),
		}
	}

	return results
}
