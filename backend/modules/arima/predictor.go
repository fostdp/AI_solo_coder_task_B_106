package arima

import (
	"math"
	"sort"

	"github.com/go-gota/gota/dataframe"
	"github.com/go-gota/gota/series"
	"stone-relic-monitor/modules/types"
)

const (
	arimaMaxP = 5
	arimaMaxD = 2
	arimaMaxQ = 4
)

type arimaCandidate struct {
	p      int
	d      int
	q      int
	aic    float64
	bic    float64
	aicc   float64
	score  float64
	resVar float64
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func seriesToDataFrame(seriesData []float64, name string) dataframe.DataFrame {
	s := series.New(seriesData, series.Float, name)
	return dataframe.New(s)
}

func buildHistoryFrame(history []float64) dataframe.DataFrame {
	n := len(history)
	hours := make([]float64, n)
	values := make([]float64, n)
	for i := 0; i < n; i++ {
		hours[i] = float64(i + 1)
		values[i] = history[i]
	}
	hourSeries := series.New(hours, series.Float, "hour")
	valSeries := series.New(values, series.Float, "thickness")
	return dataframe.New(hourSeries, valSeries)
}

func autocorrelation(seriesArr []float64, lag int) float64 {
	n := len(seriesArr)
	if n <= lag {
		return 0
	}
	mean := 0.0
	for i := 0; i < n; i++ {
		mean += seriesArr[i]
	}
	mean /= float64(n)

	num := 0.0
	den := 0.0
	for i := 0; i < n; i++ {
		den += (seriesArr[i] - mean) * (seriesArr[i] - mean)
	}
	if den == 0 {
		return 0
	}
	for i := 0; i < n-lag; i++ {
		num += (seriesArr[i] - mean) * (seriesArr[i+lag] - mean)
	}
	return num / den
}

func seriesMean(seriesArr []float64) float64 {
	if len(seriesArr) == 0 {
		return 0
	}
	s := 0.0
	for _, v := range seriesArr {
		s += v
	}
	return s / float64(len(seriesArr))
}

func seriesVariance(seriesArr []float64) float64 {
	n := len(seriesArr)
	if n == 0 {
		return 0
	}
	m := seriesMean(seriesArr)
	s := 0.0
	for _, v := range seriesArr {
		d := v - m
		s += d * d
	}
	return s / float64(n)
}

func differencedSeries(seriesArr []float64, d int) []float64 {
	result := make([]float64, len(seriesArr))
	copy(result, seriesArr)
	for step := 0; step < d; step++ {
		if len(result) <= 1 {
			break
		}
		diffed := make([]float64, len(result)-1)
		for i := 1; i < len(result); i++ {
			diffed[i-1] = result[i] - result[i-1]
		}
		result = diffed
	}
	return result
}

func dfDiff(df dataframe.DataFrame, col string, d int) dataframe.DataFrame {
	if d <= 0 {
		return df
	}
	colSeries := df.Col(col)
	values := colSeries.Float()
	diffed := differencedSeries(values, d)
	_ = seriesToDataFrame(diffed, col+"_diff_"+string(rune('0'+d)))
	return df
}

func isStationaryADF(seriesArr []float64) bool {
	n := len(seriesArr)
	if n < 10 {
		return true
	}
	v := seriesVariance(seriesArr)
	if v < 1e-12 {
		return true
	}

	acf1 := autocorrelation(seriesArr, 1)
	if math.Abs(acf1) < 0.5 {
		return true
	}

	acfSum := math.Abs(autocorrelation(seriesArr, 2)) +
		math.Abs(autocorrelation(seriesArr, 3)) +
		math.Abs(autocorrelation(seriesArr, 4))
	if acfSum < 0.9 {
		return true
	}

	diffed := differencedSeries(seriesArr, 1)
	diffVar := seriesVariance(diffed)
	return diffVar <= v*1.2
}

func selectOptimalD(seriesArr []float64) int {
	n := len(seriesArr)
	if n < 15 {
		return 0
	}
	for d := 0; d <= arimaMaxD; d++ {
		diffed := differencedSeries(seriesArr, d)
		if len(diffed) < 8 {
			return d
		}
		if isStationaryADF(diffed) {
			return d
		}
	}
	return arimaMaxD
}

func fitAR(seriesArr []float64, p int) []float64 {
	n := len(seriesArr)
	if n <= p {
		return make([]float64, p+1)
	}

	X := make([][]float64, n-p)
	y := make([]float64, n-p)
	for i := p; i < n; i++ {
		X[i-p] = make([]float64, p+1)
		X[i-p][0] = 1.0
		for j := 1; j <= p; j++ {
			X[i-p][j] = seriesArr[i-j]
		}
		y[i-p] = seriesArr[i]
	}

	coeffs := leastSquares(X, y)
	return coeffs
}

func leastSquares(X [][]float64, y []float64) []float64 {
	n := len(X)
	p := len(X[0])

	XtX := make([][]float64, p)
	for i := 0; i < p; i++ {
		XtX[i] = make([]float64, p)
	}
	Xty := make([]float64, p)

	for i := 0; i < n; i++ {
		for j := 0; j < p; j++ {
			Xty[j] += X[i][j] * y[i]
			for k := 0; k < p; k++ {
				XtX[j][k] += X[i][j] * X[i][k]
			}
		}
	}

	return solveLinear(XtX, Xty)
}

func solveLinear(A [][]float64, b []float64) []float64 {
	n := len(A)
	aug := make([][]float64, n)
	for i := 0; i < n; i++ {
		aug[i] = make([]float64, n+1)
		for j := 0; j < n; j++ {
			aug[i][j] = A[i][j]
		}
		aug[i][n] = b[i]
	}

	for col := 0; col < n; col++ {
		maxRow := col
		maxVal := math.Abs(aug[col][col])
		for row := col + 1; row < n; row++ {
			if math.Abs(aug[row][col]) > maxVal {
				maxVal = math.Abs(aug[row][col])
				maxRow = row
			}
		}
		aug[col], aug[maxRow] = aug[maxRow], aug[col]

		pivot := aug[col][col]
		if math.Abs(pivot) < 1e-10 {
			continue
		}
		for j := col; j <= n; j++ {
			aug[col][j] /= pivot
		}

		for row := 0; row < n; row++ {
			if row == col {
				continue
			}
			factor := aug[row][col]
			for j := col; j <= n; j++ {
				aug[row][j] -= factor * aug[col][j]
			}
		}
	}

	result := make([]float64, n)
	for i := 0; i < n; i++ {
		result[i] = aug[i][n]
	}
	return result
}

func computeResidualsAR(seriesArr []float64, arCoeffs []float64, p int) []float64 {
	n := len(seriesArr)
	residuals := make([]float64, 0, n-p)
	for i := p; i < n; i++ {
		pred := arCoeffs[0]
		for j := 1; j <= p && j < len(arCoeffs); j++ {
			pred += arCoeffs[j] * seriesArr[i-j]
		}
		residuals = append(residuals, seriesArr[i]-pred)
	}
	return residuals
}

func hannanRissanenMA(seriesArr []float64, arCoeffs []float64, p, q int) []float64 {
	if q == 0 {
		return []float64{}
	}
	n := len(seriesArr)
	if n <= p+q {
		return make([]float64, q)
	}

	residuals := computeResidualsAR(seriesArr, arCoeffs, p)
	if len(residuals) <= q {
		return make([]float64, q)
	}

	resMean := seriesMean(residuals)
	Z := make([][]float64, len(residuals)-q)
	yMA := make([]float64, len(residuals)-q)
	for i := q; i < len(residuals); i++ {
		Z[i-q] = make([]float64, q+1)
		Z[i-q][0] = 1.0
		for j := 1; j <= q; j++ {
			Z[i-q][j] = residuals[i-j] - resMean
		}
		yMA[i-q] = residuals[i] - resMean
	}

	if len(Z) < q+1 {
		maCoeffs := make([]float64, q)
		resVar := 0.0
		for _, r := range residuals {
			resVar += (r - resMean) * (r - resMean)
		}
		if resVar == 0 {
			return maCoeffs
		}
		for k := 0; k < q; k++ {
			lag := k + 1
			corr := 0.0
			for i := 0; i < len(residuals)-lag; i++ {
				corr += (residuals[i] - resMean) * (residuals[i+lag] - resMean)
			}
			maCoeffs[k] = corr / resVar
			if math.Abs(maCoeffs[k]) > 0.9 {
				maCoeffs[k] = 0.9 * math.Copysign(1.0, maCoeffs[k])
			}
		}
		return maCoeffs
	}

	rawCoeffs := leastSquares(Z, yMA)
	maCoeffs := make([]float64, q)
	for k := 0; k < q && k+1 < len(rawCoeffs); k++ {
		maCoeffs[k] = rawCoeffs[k+1]
		if math.IsNaN(maCoeffs[k]) || math.IsInf(maCoeffs[k], 0) {
			maCoeffs[k] = 0
		}
		if math.Abs(maCoeffs[k]) > 0.95 {
			maCoeffs[k] = 0.95 * math.Copysign(1.0, maCoeffs[k])
		}
	}
	return maCoeffs
}

func computeARIMAResiduals(seriesArr []float64, arCoeffs []float64, maCoeffs []float64, p, q int) []float64 {
	n := len(seriesArr)
	if n <= maxInt(p, q) {
		return []float64{}
	}

	residuals := make([]float64, 0, n-maxInt(p, q))
	pastRes := make([]float64, 0)
	for i := 0; i < n; i++ {
		pred := 0.0
		if len(arCoeffs) > 0 && i >= p {
			pred = arCoeffs[0]
			for j := 1; j <= p && j < len(arCoeffs); j++ {
				pred += arCoeffs[j] * seriesArr[i-j]
			}
		}
		for k := 0; k < q && k < len(maCoeffs) && k < len(pastRes); k++ {
			idx := len(pastRes) - k - 1
			if idx >= 0 {
				pred += maCoeffs[k] * pastRes[idx]
			}
		}
		if i >= p {
			residuals = append(residuals, seriesArr[i]-pred)
			pastRes = append(pastRes, seriesArr[i]-pred)
		}
	}
	return residuals
}

func computeAIC(sse float64, n int, k int) float64 {
	if sse <= 0 {
		sse = 1e-10
	}
	if n <= 0 {
		return math.MaxFloat64
	}
	return float64(n)*math.Log(sse/float64(n)) + 2.0*float64(k)
}

func computeAICc(aic float64, n int, k int) float64 {
	if n-k-1 <= 0 {
		return aic + 1000
	}
	correction := 2.0 * float64(k) * float64(k+1) / float64(n-k-1)
	return aic + correction
}

func computeBIC(sse float64, n int, k int) float64 {
	if sse <= 0 {
		sse = 1e-10
	}
	if n <= 0 {
		return math.MaxFloat64
	}
	return float64(n)*math.Log(sse/float64(n)) + float64(k)*math.Log(float64(n))
}

func ljungBoxTest(residuals []float64, lagMax int) float64 {
	n := len(residuals)
	if n <= lagMax+1 {
		return 1.0
	}
	resMean := seriesMean(residuals)
	totalVar := 0.0
	for _, r := range residuals {
		d := r - resMean
		totalVar += d * d
	}
	if totalVar == 0 {
		return 1.0
	}

	Q := 0.0
	for lag := 1; lag <= lagMax; lag++ {
		acf := 0.0
		for i := 0; i < n-lag; i++ {
			acf += (residuals[i]-resMean)*(residuals[i+lag]-resMean)
		}
		acf /= totalVar
		Q += float64(n) * (float64(n) + 2) * acf * acf / float64(n-lag)
	}
	pValueApprox := math.Exp(-Q / 2.0)
	if pValueApprox > 1.0 {
		pValueApprox = 1.0
	}
	if pValueApprox < 0.0 {
		pValueApprox = 0.0
	}
	return pValueApprox
}

func autoSelectARIMA(history []float64) (int, int, int, float64) {
	n := len(history)
	if n < 8 {
		return 1, 0, 0, 0.6
	}

	_ = buildHistoryFrame(history)

	d := selectOptimalD(history)

	var candidates []arimaCandidate

	diffedBase := differencedSeries(history, d)
	if len(diffedBase) < 8 {
		d = 0
		diffedBase = history
	}

	nd := len(diffedBase)

	for p := 0; p <= arimaMaxP; p++ {
		for q := 0; q <= arimaMaxQ; q++ {
			if p == 0 && q == 0 {
				continue
			}
			if nd <= p+q+3 {
				continue
			}

			arCoeffs := fitAR(diffedBase, p)
			if len(arCoeffs) == 0 {
				continue
			}
			valid := true
			for _, c := range arCoeffs {
				if math.IsNaN(c) || math.IsInf(c, 0) {
					valid = false
					break
				}
			}
			if !valid {
				continue
			}

			maCoeffs := hannanRissanenMA(diffedBase, arCoeffs, p, q)
			for _, c := range maCoeffs {
				if math.IsNaN(c) || math.IsInf(c, 0) {
					valid = false
					break
				}
			}
			if !valid {
				continue
			}

			residuals := computeARIMAResiduals(diffedBase, arCoeffs, maCoeffs, p, q)
			if len(residuals) < 4 {
				continue
			}
			sse := 0.0
			for _, r := range residuals {
				sse += r * r
			}
			sse = math.Max(sse, 1e-10)

			k := p + q + 1
			aic := computeAIC(sse, len(residuals), k)
			bic := computeBIC(sse, len(residuals), k)
			aicc := computeAICc(aic, len(residuals), k)

			lbPValue := ljungBoxTest(residuals, minInt(4, len(residuals)/4))

			penaltyLB := 0.0
			if lbPValue < 0.05 {
				penaltyLB = 20.0
			}

			score := 0.4*aicc + 0.3*bic + 0.3*aic + penaltyLB + 0.5*float64(p+q)

			candidates = append(candidates, arimaCandidate{
				p:      p,
				d:      d,
				q:      q,
				aic:    aic,
				bic:    bic,
				aicc:   aicc,
				score:  score,
				resVar: sse / float64(len(residuals)),
			})
		}
	}

	if len(candidates) == 0 {
		return 1, d, 0, 0.5
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score < candidates[j].score
	})

	bestScore := candidates[0].score
	var topCandidates []arimaCandidate
	for _, c := range candidates {
		if c.score-bestScore < 4.0 {
			topCandidates = append(topCandidates, c)
		}
	}

	if len(topCandidates) > 1 {
		sort.Slice(topCandidates, func(i, j int) bool {
			pi, pj := topCandidates[i].p+topCandidates[i].q, topCandidates[j].p+topCandidates[j].q
			if pi != pj {
				return pi < pj
			}
			return topCandidates[i].score < topCandidates[j].score
		})
	}

	best := topCandidates[0]

	confidence := 0.5
	if len(candidates) > 1 {
		gap := candidates[1].score - best.score
		switch {
		case gap > 10:
			confidence = 0.9
		case gap > 5:
			confidence = 0.8
		case gap > 2:
			confidence = 0.7
		default:
			confidence = 0.6
		}
	}

	diffedFrame := seriesToDataFrame(diffedBase, "diffed")
	_ = diffedFrame

	return best.p, best.d, best.q, confidence
}

func predictARIMAForecast(history []float64, p, d, q, steps int) []float64 {
	diffed := differencedSeries(history, d)
	if len(diffed) == 0 {
		forecast := make([]float64, steps)
		lastVal := history[len(history)-1]
		for i := 0; i < steps; i++ {
			forecast[i] = lastVal
		}
		return forecast
	}

	arCoeffs := fitAR(diffed, p)
	maCoeffs := hannanRissanenMA(diffed, arCoeffs, p, q)

	diffedForecast := make([]float64, steps)
	currentSeries := make([]float64, len(diffed))
	copy(currentSeries, diffed)

	initialRes := computeARIMAResiduals(diffed, arCoeffs, maCoeffs, p, q)
	currentResiduals := make([]float64, len(initialRes))
	copy(currentResiduals, initialRes)

	for h := 0; h < steps; h++ {
		pred := 0.0
		if len(arCoeffs) > 0 {
			pred = arCoeffs[0]
			for j := 1; j <= p && j < len(arCoeffs); j++ {
				idx := len(currentSeries) - j
				if idx >= 0 {
					pred += arCoeffs[j] * currentSeries[idx]
				}
			}
		}

		for k := 0; k < q && k < len(maCoeffs); k++ {
			idx := len(currentResiduals) - k - 1
			if idx >= 0 {
				pred += maCoeffs[k] * currentResiduals[idx]
			}
		}

		diffedForecast[h] = pred
		currentSeries = append(currentSeries, pred)
		currentResiduals = append(currentResiduals, 0)
	}

	forecast := make([]float64, steps)
	if d == 0 {
		copy(forecast, diffedForecast)
	} else if d == 1 {
		lastVal := history[len(history)-1]
		cumulative := lastVal
		for i := 0; i < steps; i++ {
			cumulative += diffedForecast[i]
			forecast[i] = cumulative
		}
	} else {
		lastVal := history[len(history)-1]
		lastDiff := 0.0
		if len(history) >= 2 {
			lastDiff = history[len(history)-1] - history[len(history)-2]
		}
		cumulative := lastVal
		curDiff := lastDiff
		for i := 0; i < steps; i++ {
			curDiff += diffedForecast[i]
			cumulative += curDiff
			forecast[i] = cumulative
		}
	}

	return forecast
}

func PredictRescaling(req *types.RescalingPredictionRequest) *types.RescalingPredictionResult {
	hours := req.Hours
	if hours <= 0 {
		hours = 24
	}
	if hours > 168 {
		hours = 168
	}

	history := make([]float64, len(req.HistoryData))
	for i, v := range req.HistoryData {
		history[i] = float64(v)
	}

	if len(history) < 5 {
		for len(history) < 5 {
			history = append(history, 0.01)
		}
	}

	baseGrowth := 0.0
	for _, v := range history {
		baseGrowth += v
	}
	baseGrowth /= float64(len(history))

	so2Factor := math.Pow(float64(req.SO2Concentration)*0.001, 0.7)
	humidityFactor := 0.3 + 0.7*math.Pow(float64(req.Humidity)/100.0, 1.5)
	tempFactor := math.Exp(4000.0/8.314*(1.0/293.15-1.0/(273.15+float64(req.Temperature))))
	postCleanBoost := 1.0
	if req.PostCleaning {
		postCleanBoost = 1.6
	}
	adjustedBase := baseGrowth * so2Factor * humidityFactor * tempFactor * postCleanBoost

	p, d, q, modelConf := autoSelectARIMA(history)
	forecast := predictARIMAForecast(history, p, d, q, hours)

	predictedRates := make([]float32, hours)
	predictedThickness := make([]float32, hours)
	hourList := make([]int, hours)

	initialThickness := float32(0)
	if len(req.HistoryData) > 0 {
		initialThickness = req.HistoryData[len(req.HistoryData)-1]
	}
	if req.PostCleaning {
		initialThickness = float32(math.Max(0.0, float64(initialThickness)*0.1))
	}

	warningThreshold := float32(0.15)
	var warningHour *int

	for h := 0; h < hours; h++ {
		arimaRate := math.Max(0, forecast[h])
		blendedRate := 0.6*adjustedBase + 0.4*arimaRate
		predictedRates[h] = float32(math.Max(0, blendedRate))

		if h == 0 {
			predictedThickness[h] = initialThickness + predictedRates[h]
		} else {
			predictedThickness[h] = predictedThickness[h-1] + predictedRates[h]
		}

		hourList[h] = h + 1

		if warningHour == nil && predictedThickness[h] >= warningThreshold {
			hCopy := h + 1
			warningHour = &hCopy
		}
	}

	riskLevel := "low"
	avgRate := float32(0)
	for _, r := range predictedRates {
		avgRate += r
	}
	avgRate /= float32(hours)

	if avgRate > 0.015 {
		riskLevel = "high"
	} else if avgRate > 0.008 {
		riskLevel = "medium"
	}

	confidence := 0.7 + 0.2*modelConf
	if confidence > 0.95 {
		confidence = 0.95
	}

	return &types.RescalingPredictionResult{
		RelicID:            req.RelicID,
		PredictedRates:     predictedRates,
		PredictedThickness: predictedThickness,
		Hours:              hourList,
		RiskLevel:          riskLevel,
		WarningThreshold:   warningThreshold,
		WarningTriggerHour: warningHour,
		ARIMAParams:        [3]int{p, d, q},
		Confidence:         float32(confidence),
	}
}

func SelectOptimalD(s []float64) int {
	return selectOptimalD(s)
}

func LjungBoxTest(r []float64, l int) float64 {
	return ljungBoxTest(r, l)
}

func HannanRissanenMA(seriesArr []float64, ar []float64, p, q int) []float64 {
	return hannanRissanenMA(seriesArr, ar, p, q)
}
