package tsp

import (
	"context"
	"math"
	"math/rand"
	"sort"
	"sync"
	"time"

	"gonum.org/v1/gonum/optimize"
	"stone-relic-monitor/modules/types"
)

const (
	smallThreshold  = 20
	mediumThreshold = 50
	timeoutMs       = 500
)

type Solver struct {
	mu         sync.RWMutex
	lastResult *types.TSPPathResult
}

func NewSolver() *Solver {
	return &Solver{}
}

func euclideanDistance3D(a, b *types.CleaningPoint) float64 {
	dx := float64(a.X - b.X)
	dy := float64(a.Y - b.Y)
	dz := float64(a.Z - b.Z)
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

func buildDistanceMatrix(points []types.CleaningPoint) [][]float64 {
	n := len(points)
	dist := make([][]float64, n)
	for i := 0; i < n; i++ {
		dist[i] = make([]float64, n)
		for j := 0; j < n; j++ {
			if i == j {
				dist[i][j] = 0
			} else {
				dist[i][j] = euclideanDistance3D(&points[i], &points[j])
			}
		}
	}
	return dist
}

func pathLength(dist [][]float64, order []int) float64 {
	if len(order) < 2 {
		return 0
	}
	total := 0.0
	for i := 0; i < len(order)-1; i++ {
		total += dist[order[i]][order[i+1]]
	}
	return total
}

func primMST(dist [][]float64, start int) []int {
	n := len(dist)
	parent := make([]int, n)
	key := make([]float64, n)
	inMST := make([]bool, n)
	for i := 0; i < n; i++ {
		key[i] = math.MaxFloat64
		parent[i] = -1
	}
	key[start] = 0

	for count := 0; count < n-1; count++ {
		u := -1
		minVal := math.MaxFloat64
		for v := 0; v < n; v++ {
			if !inMST[v] && key[v] < minVal {
				minVal = key[v]
				u = v
			}
		}
		if u == -1 {
			break
		}
		inMST[u] = true
		for v := 0; v < n; v++ {
			if !inMST[v] && dist[u][v] > 0 && dist[u][v] < key[v] {
				parent[v] = u
				key[v] = dist[u][v]
			}
		}
	}
	return parent
}

func findOddDegreeVertices(parent []int, dist [][]float64) []int {
	n := len(parent)
	degree := make([]int, n)
	for v := 0; v < n; v++ {
		if parent[v] != -1 {
			degree[v]++
			degree[parent[v]]++
		}
	}
	var odds []int
	for v := 0; v < n; v++ {
		if degree[v]%2 != 0 {
			odds = append(odds, v)
		}
	}
	return odds
}

func greedyPerfectMatching(odds []int, dist [][]float64) map[int]int {
	matching := make(map[int]int)
	used := make([]bool, len(odds))
	for i := 0; i < len(odds); i++ {
		if used[i] {
			continue
		}
		bestJ := -1
		bestD := math.MaxFloat64
		for j := i + 1; j < len(odds); j++ {
			if !used[j] && dist[odds[i]][odds[j]] < bestD {
				bestD = dist[odds[i]][odds[j]]
				bestJ = j
			}
		}
		if bestJ >= 0 {
			used[i] = true
			used[bestJ] = true
			matching[odds[i]] = odds[bestJ]
			matching[odds[bestJ]] = odds[i]
		}
	}
	return matching
}

func buildMultigraph(parent []int, matching map[int]int, n int) [][]int {
	adj := make([][]int, n)
	for v := 0; v < n; v++ {
		if parent[v] != -1 {
			adj[v] = append(adj[v], parent[v])
			adj[parent[v]] = append(adj[parent[v]], v)
		}
	}
	for u, v := range matching {
		if u < v {
			adj[u] = append(adj[u], v)
			adj[v] = append(adj[v], u)
		}
	}
	return adj
}

func hierholzerEulerCircuit(adj [][]int, start int) []int {
	n := len(adj)
	working := make([][]int, n)
	for i := 0; i < n; i++ {
		working[i] = make([]int, len(adj[i]))
		copy(working[i], adj[i])
	}

	stack := []int{start}
	var circuit []int
	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		if len(working[cur]) > 0 {
			next := working[cur][len(working[cur])-1]
			working[cur] = working[cur][:len(working[cur])-1]
			for i, v := range working[next] {
				if v == cur {
					working[next] = append(working[next][:i], working[next][i+1:]...)
					break
				}
			}
			stack = append(stack, next)
		} else {
			circuit = append(circuit, cur)
			stack = stack[:len(stack)-1]
		}
	}
	for i, j := 0, len(circuit)-1; i < j; i, j = i+1, j-1 {
		circuit[i], circuit[j] = circuit[j], circuit[i]
	}
	return circuit
}

func shortcutHamiltonian(euler []int) []int {
	visited := make(map[int]bool)
	var ham []int
	for _, v := range euler {
		if !visited[v] {
			visited[v] = true
			ham = append(ham, v)
		}
	}
	return ham
}

func christofidesTSP(points []types.CleaningPoint, startIdx int, dist [][]float64) []int {
	n := len(points)
	if n == 0 {
		return []int{}
	}
	if n == 1 {
		return []int{0}
	}
	start := 0
	if startIdx >= 0 && startIdx < n {
		start = startIdx
	}
	parent := primMST(dist, start)
	odds := findOddDegreeVertices(parent, dist)
	matching := greedyPerfectMatching(odds, dist)
	adj := buildMultigraph(parent, matching, n)
	euler := hierholzerEulerCircuit(adj, start)
	ham := shortcutHamiltonian(euler)
	if len(ham) != n {
		ham = make([]int, n)
		for i := 0; i < n; i++ {
			ham[i] = i
		}
	}
	return ham
}

func orOptTSP(dist [][]float64, order []int, deadline time.Time, maxIters int) []int {
	n := len(order)
	if n < 4 {
		return order
	}
	improved := true
	iters := 0
	for improved && iters < maxIters && time.Now().Before(deadline) {
		improved = false
		iters++
		for segLen := 1; segLen <= 3; segLen++ {
			for i := 0; i <= n-segLen; i++ {
				for j := 0; j <= n; j++ {
					if j >= i && j < i+segLen {
						continue
					}
					if time.Now().After(deadline) {
						return order
					}
					a, b, c, d := -1, -1, -1, -1
					if i > 0 {
						a = order[i-1]
					}
					b = order[i]
					c = order[i+segLen-1]
					if i+segLen < n {
						d = order[i+segLen]
					}
					removeCost := 0.0
					if a >= 0 {
						removeCost += dist[a][b]
					}
					if d >= 0 {
						removeCost += dist[c][d]
					}
					if a >= 0 && d >= 0 {
						removeCost -= dist[a][d]
					}

					insertCost := 0.0
					x, y := -1, -1
					if j == 0 {
						y = order[0]
					} else if j == n {
						x = order[n-1]
					} else {
						x = order[j-1]
						y = order[j]
					}
					if x >= 0 {
						insertCost += dist[x][b]
					}
					if y >= 0 {
						insertCost += dist[c][y]
					}
					if x >= 0 && y >= 0 {
						insertCost -= dist[x][y]
					}

					if insertCost < removeCost-1e-9 {
						segment := make([]int, segLen)
						copy(segment, order[i:i+segLen])
						newOrder := make([]int, 0, n)
						newOrder = append(newOrder, order[:i]...)
						newOrder = append(newOrder, order[i+segLen:]...)
						insertPos := j
						if j > i+segLen {
							insertPos = j - segLen
						}
						temp := make([]int, 0, n)
						temp = append(temp, newOrder[:insertPos]...)
						temp = append(temp, segment...)
						temp = append(temp, newOrder[insertPos:]...)
						order = temp
						improved = true
					}
				}
			}
		}
	}
	return order
}

func twoOptOptimization(dist [][]float64, order []int, deadline time.Time, maxIters int) []int {
	n := len(order)
	if n < 4 {
		return order
	}
	improved := true
	iters := 0
	for improved && iters < maxIters && time.Now().Before(deadline) {
		improved = false
		iters++
		for i := 0; i < n-1 && time.Now().Before(deadline); i++ {
			for k := i + 1; k < n; k++ {
				a, b := -1, -1
				if i > 0 {
					a = order[i-1]
				}
				b = order[i]
				c := order[k]
				d := -1
				if k+1 < n {
					d = order[k+1]
				}
				delta := 0.0
				if a >= 0 {
					delta += -dist[a][b] + dist[a][c]
				}
				if d >= 0 {
					delta += -dist[c][d] + dist[b][d]
				}
				if delta < -1e-9 {
					for p, q := i, k; p < q; p, q = p+1, q-1 {
						order[p], order[q] = order[q], order[p]
					}
					improved = true
				}
			}
		}
	}
	return order
}

func nearestNeighborTSP(dist [][]float64, start int) []int {
	n := len(dist)
	if n == 0 {
		return []int{}
	}
	visited := make([]bool, n)
	order := make([]int, 0, n)
	cur := start
	visited[cur] = true
	order = append(order, cur)
	for len(order) < n {
		best := -1
		bestD := math.MaxFloat64
		for j := 0; j < n; j++ {
			if !visited[j] && dist[cur][j] < bestD {
				bestD = dist[cur][j]
				best = j
			}
		}
		if best < 0 {
			break
		}
		visited[best] = true
		order = append(order, best)
		cur = best
	}
	return order
}

type randomKeyTSP struct {
	dist   [][]float64
	points []types.CleaningPoint
}

func (r *randomKeyTSP) Func(x []float64) float64 {
	n := len(x)
	type pair struct {
		key   float64
		index int
	}
	pairs := make([]pair, n)
	for i, v := range x {
		pairs[i] = pair{v, i}
	}
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].key < pairs[j].key
	})
	order := make([]int, n)
	for i, p := range pairs {
		order[i] = p.index
	}
	return pathLength(r.dist, order)
}

func optimizeWithGonum(dist [][]float64, points []types.CleaningPoint, initOrder []int, deadline time.Time) []int {
	n := len(points)
	if n < 4 {
		return initOrder
	}

	rk := &randomKeyTSP{dist: dist, points: points}

	initX := make([]float64, n)
	invOrder := make([]int, n)
	for i, v := range initOrder {
		invOrder[v] = i
	}
	for i := 0; i < n; i++ {
		initX[i] = float64(invOrder[i])/float64(n) + 0.001
		if i%2 == 0 {
			initX[i] += 0.0005
		}
	}

	problem := optimize.Problem{
		Func: rk.Func,
	}

	timeoutDur := time.Until(deadline)
	if timeoutDur < 50*time.Millisecond {
		return initOrder
	}

	settings := &optimize.Settings{
		MajorIterations: 30,
		FuncEvaluations:  200,
		Runtime:          timeoutDur,
		Concurrent:       1,
	}

	locOpt := &optimize.NelderMead{
		SimplexSize: 0.05,
	}

	result, err := optimize.Minimize(problem, initX, settings, locOpt)
	if err != nil || result == nil || len(result.Location.X) != n {
		return initOrder
	}

	type pair struct {
		key   float64
		index int
	}
	pairs := make([]pair, n)
	for i, v := range result.Location.X {
		pairs[i] = pair{v, i}
	}
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].key < pairs[j].key
	})
	order := make([]int, n)
	for i, p := range pairs {
		order[i] = p.index
	}

	lenOpt := pathLength(dist, order)
	lenInit := pathLength(dist, initOrder)
	if lenOpt < lenInit {
		return order
	}
	return initOrder
}

func SolveTSP(ctx context.Context, req *types.TSPPathRequest) *types.TSPPathResult {
	return NewSolver().Solve(ctx, req)
}

func (s *Solver) Solve(ctx context.Context, req *types.TSPPathRequest) *types.TSPPathResult {
	if req == nil || len(req.Points) == 0 {
		return &types.TSPPathResult{
			RelicID:       req.RelicID,
			OrderedPoints: []types.CleaningPoint{},
			PathIndices:   []int{},
			Algorithm:     "empty",
		}
	}

	n := len(req.Points)
	dist := buildDistanceMatrix(req.Points)

	startIdx := 0
	if req.StartPoint != nil {
		bestIdx := 0
		bestD := math.MaxFloat64
		for i, p := range req.Points {
			d := euclideanDistance3D(req.StartPoint, &p)
			if d < bestD {
				bestD = d
				bestIdx = i
			}
		}
		startIdx = bestIdx
	}

	resultCh := make(chan *types.TSPPathResult, 1)
	deadline := time.Now().Add(timeoutMs * time.Millisecond)

	go func() {
		var order []int
		algorithm := ""
		iterations := 0

		switch req.Algorithm {
		case "priority":
			order = prioritySortedOrder(req.Points)
			algorithm = "priority"
		case "nearest":
			order = nearestNeighborTSP(dist, startIdx)
			algorithm = "nearest"
		case "random":
			order = make([]int, n)
			for i := 0; i < n; i++ {
				order[i] = i
			}
			rand.Shuffle(n, func(i, j int) { order[i], order[j] = order[j], order[i] })
			algorithm = "random"
		case "christofides":
			order = christofidesTSP(req.Points, startIdx, dist)
			if n <= mediumThreshold {
				order = twoOptOptimization(dist, order, deadline, 10)
				iterations = 10
			}
			algorithm = "christofides"
		case "two_opt", "tsp":
			initial := nearestNeighborTSP(dist, startIdx)
			if n <= smallThreshold {
				order = twoOptOptimization(dist, initial, deadline, 50)
				iterations = 50
			} else {
				order = twoOptOptimization(dist, initial, deadline, 20)
				iterations = 20
			}
			algorithm = "two_opt"
		default:
			switch {
			case n <= smallThreshold:
				order = christofidesTSP(req.Points, startIdx, dist)
				order = twoOptOptimization(dist, order, deadline, 50)
				algorithm = "christofides+2opt"
				iterations = 50
			case n <= mediumThreshold:
				order = christofidesTSP(req.Points, startIdx, dist)
				order = twoOptOptimization(dist, order, deadline, 20)
				algorithm = "christofides+2opt"
				iterations = 20
			default:
				order = christofidesTSP(req.Points, startIdx, dist)
				order = orOptTSP(dist, order, deadline, 3)
				order = twoOptOptimization(dist, order, deadline, 5)
				if time.Now().Before(deadline.Add(-80 * time.Millisecond)) {
					order = optimizeWithGonum(dist, req.Points, order, deadline.Add(-20*time.Millisecond))
				}
				algorithm = "christofides+oropt+2opt+gonum"
				iterations = 8
			}
		}

		ordered := make([]types.CleaningPoint, n)
		indices := make([]int, n)
		for i, idx := range order {
			if idx >= 0 && idx < n {
				ordered[i] = req.Points[idx]
				indices[i] = idx
			}
		}

		totalDist := float32(pathLength(dist, order))
		speed := float32(50.0)
		if req.RobotSpeed > 0 {
			speed = req.RobotSpeed
		}
		totalTime := float32(0)
		if speed > 0 {
			totalTime = totalDist / speed
		}

		res := &types.TSPPathResult{
			RelicID:          req.RelicID,
			OrderedPoints:    ordered,
			TotalDistance:    totalDist,
			TotalTimeSeconds: totalTime,
			PathIndices:      indices,
			Algorithm:        algorithm,
			Iterations:       iterations,
		}

		select {
		case resultCh <- res:
		default:
		}
	}()

	select {
	case res := <-resultCh:
		s.mu.Lock()
		s.lastResult = res
		s.mu.Unlock()
		return res
	case <-ctx.Done():
		fallback := nearestNeighborFallback(dist, startIdx)
		ordered := make([]types.CleaningPoint, n)
		indices := make([]int, n)
		for i, idx := range fallback {
			if idx >= 0 && idx < n {
				ordered[i] = req.Points[idx]
				indices[i] = idx
			}
		}
		return &types.TSPPathResult{
			RelicID:          req.RelicID,
			OrderedPoints:    ordered,
			TotalDistance:    float32(pathLength(dist, fallback)),
			TotalTimeSeconds: float32(pathLength(dist, fallback)) / float32(math.Max(1, float64(req.RobotSpeed))),
			PathIndices:      indices,
			Algorithm:        "nearest_neighbor_fallback",
			Iterations:       1,
		}
	case <-time.After(timeoutMs * 2 * time.Millisecond):
		fallback := nearestNeighborFallback(dist, startIdx)
		ordered := make([]types.CleaningPoint, n)
		indices := make([]int, n)
		for i, idx := range fallback {
			if idx >= 0 && idx < n {
				ordered[i] = req.Points[idx]
				indices[i] = idx
			}
		}
		return &types.TSPPathResult{
			RelicID:          req.RelicID,
			OrderedPoints:    ordered,
			TotalDistance:    float32(pathLength(dist, fallback)),
			TotalTimeSeconds: float32(pathLength(dist, fallback)) / float32(math.Max(1, float64(req.RobotSpeed))),
			PathIndices:      indices,
			Algorithm:        "nearest_neighbor_timeout",
			Iterations:       1,
		}
	}
}

func nearestNeighborFallback(dist [][]float64, start int) []int {
	return nearestNeighborTSP(dist, start)
}

func prioritySortedOrder(points []types.CleaningPoint) []int {
	n := len(points)
	indices := make([]int, n)
	for i := 0; i < n; i++ {
		indices[i] = i
	}
	for i := 0; i < n-1; i++ {
		for j := i + 1; j < n; j++ {
			pi, pj := points[indices[i]].Priority, points[indices[j]].Priority
			ti, tj := points[indices[i]].Thickness, points[indices[j]].Thickness
			if pj > pi || (pj == pi && tj > ti) {
				indices[i], indices[j] = indices[j], indices[i]
			}
		}
	}
	return indices
}
