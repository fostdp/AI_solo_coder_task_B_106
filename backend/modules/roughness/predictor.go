package roughness

import (
	"math"
	"math/rand"
	"sort"
	"sync"

	"stone-relic-monitor/modules/types"
)

const (
	nTrees       = 60
	maxDepth     = 14
	minLeafSize  = 4
	nFeatures    = 12
	nTrainSample = 800
)

type node struct {
	featureIndex int
	threshold    float64
	left         *node
	right        *node
	prediction   float64
	isLeaf       bool
	samples      int
}

type tree struct {
	root *node
}

type forest struct {
	trees          []*tree
	featureNames   []string
	trained        bool
	trainingMutex  sync.Mutex
	oobPredictions []float64
}

var (
	globalForest *forest
	forestOnce   sync.Once
)

func materialAblationThreshold(cs, cc, dol, sil float64) float64 {
	return 1.2*cs + 1.8*cc + 2.4*dol + 3.5*sil
}

func physicsBasedRoughness(energyDensity, laserPower, pulseDuration, scanSpeed,
	initialRoughness, overlapRate, cs, cc, dol, sil float64) float64 {
	fth := materialAblationThreshold(cs, cc, dol, sil)
	ratio := energyDensity / math.Max(fth, 0.1)

	var ablationEff float64
	switch {
	case ratio < 0.5:
		ablationEff = 2.0 * ratio * ratio
	case ratio < 1.0:
		ablationEff = 0.5 + 1.0*(ratio-0.5)
	case ratio < 3.0:
		ablationEff = 1.0 + 0.3*math.Log(ratio)
	default:
		ablationEff = 1.33 + 0.02*(ratio-3.0)
	}
	ablationEff = math.Max(0, math.Min(ablationEff, 1.4))

	pulseEnergy := laserPower * pulseDuration / 1000.0
	couplingFactor := overlapRate * pulseEnergy / math.Max(scanSpeed, 1.0)
	couplingFactor = math.Min(couplingFactor, 1.2)

	thermalDamage := 0.0
	if ratio > 2.0 {
		thermalDamage = 0.08 * (ratio - 2.0)
	}

	effectiveRa := initialRoughness * (1.0 - ablationEff*math.Min(0.7, couplingFactor))
	effectiveRa += initialRoughness * thermalDamage

	return effectiveRa
}

func physicsBlendWeight(energyDensity float64) float64 {
	switch {
	case energyDensity <= 0.6:
		return 0.85
	case energyDensity <= 1.0:
		return 0.85 - 0.55*(energyDensity-0.6)/0.4
	case energyDensity <= 2.0:
		return 0.30 - 0.15*(energyDensity-1.0)/1.0
	default:
		return 0.10
	}
}

func sampleVariance(vals []float64) float64 {
	n := len(vals)
	if n < 2 {
		return 0
	}
	m := 0.0
	for _, v := range vals {
		m += v
	}
	m /= float64(n)
	s := 0.0
	for _, v := range vals {
		d := v - m
		s += d * d
	}
	return s / float64(n)
}

func (t *tree) train(data [][]float64, targets []float64, sampleIdx []int, depth int, featureSubsetSize int) {
	t.root = buildTree(data, targets, sampleIdx, depth, featureSubsetSize)
}

func buildTree(data [][]float64, targets []float64, sampleIdx []int, depth int, featSubset int) *node {
	n := len(sampleIdx)
	if n == 0 {
		return &node{isLeaf: true, prediction: 0, samples: 0}
	}

	targetVals := make([]float64, n)
	for i, idx := range sampleIdx {
		targetVals[i] = targets[idx]
	}

	pred := 0.0
	for _, v := range targetVals {
		pred += v
	}
	pred /= float64(n)

	if n < minLeafSize || depth >= maxDepth || sampleVariance(targetVals) < 1e-6 {
		return &node{isLeaf: true, prediction: pred, samples: n}
	}

	nFeats := len(data[0])
	feats := make([]int, nFeats)
	for i := 0; i < nFeats; i++ {
		feats[i] = i
	}
	rand.Shuffle(len(feats), func(i, j int) { feats[i], feats[j] = feats[j], feats[i] })
	k := featSubset
	if k > len(feats) {
		k = len(feats)
	}
	candidateFeats := feats[:k]

	bestFeat := -1
	bestThresh := 0.0
	bestScore := math.MaxFloat64
	var bestLeftIdx, bestRightIdx []int

	for _, fIdx := range candidateFeats {
		vals := make([]float64, n)
		for i, s := range sampleIdx {
			vals[i] = data[s][fIdx]
		}

		sortedPairs := make([]struct {
			v   float64
			idx int
			orig int
		}, n)
		for i := 0; i < n; i++ {
			sortedPairs[i] = struct {
				v   float64
				idx int
				orig int
			}{vals[i], i, sampleIdx[i]}
		}
		sort.Slice(sortedPairs, func(i, j int) bool {
			return sortedPairs[i].v < sortedPairs[j].v
		})

		step := n / 10
		if step < 1 {
			step = 1
		}
		for sp := step; sp < n-step; sp += step {
			thresh := (sortedPairs[sp].v + sortedPairs[sp+1].v) / 2.0
			if sp-1 >= 0 && sp < n {
				thresh = (sortedPairs[sp-1].v + sortedPairs[sp].v) / 2.0
			}

			var leftVals, rightVals []float64
			var li, ri []int
			for _, pair := range sortedPairs {
				if pair.v <= thresh {
					leftVals = append(leftVals, targets[pair.orig])
					li = append(li, pair.orig)
				} else {
					rightVals = append(rightVals, targets[pair.orig])
					ri = append(ri, pair.orig)
				}
			}

			if len(li) < minLeafSize/2 || len(ri) < minLeafSize/2 {
				continue
			}

			varLeft := sampleVariance(leftVals) * float64(len(leftVals))
			varRight := sampleVariance(rightVals) * float64(len(rightVals))
			score := (varLeft + varRight) / float64(n)

			if score < bestScore {
				bestScore = score
				bestFeat = fIdx
				bestThresh = thresh
				bestLeftIdx = li
				bestRightIdx = ri
			}
		}
	}

	if bestFeat < 0 || len(bestLeftIdx) == 0 || len(bestRightIdx) == 0 {
		return &node{isLeaf: true, prediction: pred, samples: n}
	}

	return &node{
		featureIndex: bestFeat,
		threshold:    bestThresh,
		left:         buildTree(data, targets, bestLeftIdx, depth+1, featSubset),
		right:        buildTree(data, targets, bestRightIdx, depth+1, featSubset),
		prediction:   pred,
		samples:      n,
	}
}

func (t *tree) predict(features []float64) float64 {
	cur := t.root
	if cur == nil {
		return 0
	}
	for !cur.isLeaf {
		if features[cur.featureIndex] <= cur.threshold {
			cur = cur.left
		} else {
			cur = cur.right
		}
		if cur == nil {
			return 0
		}
	}
	return cur.prediction
}

func buildTrainingSet() ([][]float64, []float64, []string) {
	rand.Seed(42)
	data := make([][]float64, nTrainSample)
	targets := make([]float64, nTrainSample)

	for i := 0; i < nTrainSample; i++ {
		var energyDensity float64
		noiseScale := 0.1
		if i < 200 {
			energyDensity = 0.3 + rand.Float64()*1.2
			noiseScale = 0.05
		} else {
			energyDensity = 0.5 + rand.Float64()*4.5
		}
		laserPower := 50 + rand.Float64()*250
		pulseDuration := 200 + rand.Float64()*1800
		scanSpeed := 10 + rand.Float64()*190
		initialRoughness := 5 + rand.Float64()*45
		overlapRate := 0.1 + rand.Float64()*0.8

		cs := rand.Float64() * 0.8
		cc := rand.Float64() * (1 - cs)
		dol := rand.Float64() * (1 - cs - cc)
		sil := 1 - cs - cc - dol
		if sil < 0 {
			sil = 0
		}

		features := make([]float64, nFeatures)
		features[0] = energyDensity
		features[1] = laserPower
		features[2] = pulseDuration
		features[3] = scanSpeed
		features[4] = initialRoughness
		features[5] = overlapRate
		features[6] = cs
		features[7] = cc
		features[8] = dol
		features[9] = sil
		features[10] = energyDensity * energyDensity
		features[11] = laserPower / math.Max(scanSpeed, 1.0)

		physRa := physicsBasedRoughness(energyDensity, laserPower, pulseDuration, scanSpeed,
			initialRoughness, overlapRate, cs, cc, dol, sil)

		blendW := physicsBlendWeight(energyDensity)
		dataTerm := physRa * (1 + noiseScale*(rand.NormFloat64()*0.3))
		targets[i] = blendW*physRa + (1-blendW)*dataTerm
		targets[i] = math.Max(initialRoughness*0.3, math.Min(initialRoughness*1.05, targets[i]))
		data[i] = features
	}

	featNames := []string{
		"energy_density", "laser_power", "pulse_duration", "scan_speed",
		"initial_roughness", "overlap_rate", "calcium_sulfate", "calcite",
		"dolomite", "silicate", "energy_squared", "power_speed_ratio",
	}

	return data, targets, featNames
}

func bootstrapSample(n int) ([]int, []int) {
	inBag := make([]int, 0, n)
	oobIdx := make([]bool, n)
	for i := 0; i < n; i++ {
		idx := rand.Intn(n)
		inBag = append(inBag, idx)
		oobIdx[idx] = true
	}
	var oob []int
	for i := 0; i < n; i++ {
		if !oobIdx[i] {
			oob = append(oob, i)
		}
	}
	return inBag, oob
}

func trainForest() *forest {
	data, targets, names := buildTrainingSet()
	n := len(data)
	featSubset := int(math.Sqrt(float64(nFeatures)))

	f := &forest{
		trees:        make([]*tree, nTrees),
		featureNames: names,
		trained:      true,
	}

	for ti := 0; ti < nTrees; ti++ {
		inBag, _ := bootstrapSample(n)
		tr := &tree{}
		tr.train(data, targets, inBag, 0, featSubset)
		f.trees[ti] = tr
	}

	return f
}

func ensureForestTrained() *forest {
	forestOnce.Do(func() {
		globalForest = trainForest()
	})
	return globalForest
}

func extractFeatures(req *types.RoughnessPredictionRequest) ([]float64, float64, float64, float64, float64, float64, float64, float64, float64, float64, float64) {
	cs := float64(req.MineralComposition["calcium_sulfate"])
	cc := float64(req.MineralComposition["calcite"])
	dol := float64(req.MineralComposition["dolomite"])
	sil := float64(req.MineralComposition["silicate"])

	energy := float64(req.EnergyDensity)
	power := float64(req.LaserPower)
	pulse := float64(req.PulseDuration)
	speed := float64(req.ScanSpeed)
	initRough := float64(req.InitialRoughness)
	overlap := float64(req.OverlapRate)

	features := make([]float64, nFeatures)
	features[0] = energy
	features[1] = power
	features[2] = pulse
	features[3] = speed
	features[4] = initRough
	features[5] = overlap
	features[6] = cs
	features[7] = cc
	features[8] = dol
	features[9] = sil
	features[10] = energy * energy
	features[11] = power / math.Max(speed, 1.0)

	return features, energy, power, pulse, speed, initRough, overlap, cs, cc, dol, sil
}

func PredictStd(f *forest, features []float64) float64 {
	nTr := len(f.trees)
	preds := make([]float64, nTr)
	for i, t := range f.trees {
		preds[i] = t.predict(features)
	}
	m := 0.0
	for _, p := range preds {
		m += p
	}
	m /= float64(nTr)
	variance := 0.0
	for _, p := range preds {
		d := p - m
		variance += d * d
	}
	variance /= float64(nTr - 1)
	return math.Sqrt(variance)
}

func PredictRoughness(req *types.RoughnessPredictionRequest) *types.RoughnessPredictionResult {
	if req.MineralComposition == nil {
		req.MineralComposition = map[string]float32{
			"calcium_sulfate": 0.6,
			"calcite":         0.25,
			"dolomite":        0.1,
			"silicate":        0.05,
		}
	}

	features, energy, power, pulse, speed, initRough, overlap,
		cs, cc, dol, sil := extractFeatures(req)

	f := ensureForestTrained()

	rfPred := 0.0
	for _, t := range f.trees {
		rfPred += t.predict(features)
	}
	rfPred /= float64(len(f.trees))

	physPred := physicsBasedRoughness(energy, power, pulse, speed,
		initRough, overlap, cs, cc, dol, sil)

	w := physicsBlendWeight(energy)
	blended := w*physPred + (1-w)*rfPred

	lowerBound := initRough * 0.3
	upperBound := initRough * 1.05
	if blended < lowerBound {
		blended = lowerBound
	}
	if blended > upperBound {
		blended = upperBound
	}

	predStd := PredictStd(f, features)

	featureImportance := computeFeatureImportance(f, features)

	riskLevel := "low"
	raNorm := blended / math.Max(initRough, 0.1)
	if raNorm > 0.9 {
		riskLevel = "high"
	} else if raNorm > 0.6 {
		riskLevel = "medium"
	}

	confidence := float32(0.85)
	if predStd > blended*0.2 {
		confidence = 0.65
	}
	if energy < 1.0 {
		confidence += 0.05
	}

	return &types.RoughnessPredictionResult{
		RelicID:            req.RelicID,
		PredictedRoughness: float32(blended),
		Confidence:         confidence,
		FeatureImportance:  featureImportance,
		RoughnessRange:     [2]float32{float32(math.Max(0.1, blended-2*predStd)), float32(blended + 2*predStd)},
		RiskLevel:          riskLevel,
	}
}

func computeFeatureImportance(f *forest, features []float64) map[string]float32 {
	importance := make(map[string]float32)
	if f == nil || len(f.trees) == 0 {
		return importance
	}

	basePred := 0.0
	for _, t := range f.trees {
		basePred += t.predict(features)
	}
	basePred /= float64(len(f.trees))

	for fi, name := range f.featureNames {
		origVal := features[fi]
		delta := 0.0
		if origVal != 0 {
			delta = math.Abs(origVal) * 0.1
		} else {
			delta = 0.01
		}

		permuted := make([]float64, len(features))
		copy(permuted, features)
		permuted[fi] = origVal + delta

		permPred := 0.0
		for _, t := range f.trees {
			permPred += t.predict(permuted)
		}
		permPred /= float64(len(f.trees))

		score := math.Abs(permPred - basePred) / math.Max(delta, 1e-6)
		importance[name] = float32(score)
	}

	total := float32(0)
	for _, v := range importance {
		total += v
	}
	if total > 0 {
		for k, v := range importance {
			importance[k] = v / total
		}
	}
	return importance
}

func MaterialAblationThreshold(cs, cc, dol, sil float64) float64 {
	return materialAblationThreshold(cs, cc, dol, sil)
}

func PhysicsBasedRoughness(energyDensity, laserPower, pulseDuration, scanSpeed,
	initialRoughness, overlapRate, cs, cc, dol, sil float64) float64 {
	return physicsBasedRoughness(energyDensity, laserPower, pulseDuration, scanSpeed,
		initialRoughness, overlapRate, cs, cc, dol, sil)
}

func PhysicsBlendWeight(energyDensity float64) float64 {
	return physicsBlendWeight(energyDensity)
}
