package handlers

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"stone-relic-monitor/internal/config"
	"stone-relic-monitor/modules/types"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupAdvancedHandler() (*AdvancedCleaningHandler, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{}
	handler := NewAdvancedCleaningHandler(cfg, nil)
	r := gin.New()
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return handler, r
}

func generateHandlerTestPoints(n int) []types.CleaningPoint {
	points := make([]types.CleaningPoint, n)
	for i := 0; i < n; i++ {
		points[i] = types.CleaningPoint{
			ID:        i,
			X:         rand.Float32() * 100,
			Y:         rand.Float32() * 20,
			Z:         rand.Float32() * 100,
			Thickness: 0.5 + rand.Float32()*3.5,
			Area:      1.0 + rand.Float32()*2.0,
			Priority:  rand.Intn(3) + 1,
		}
	}
	return points
}

func TestPlanTSPPath_Handler(t *testing.T) {
	_, r := setupAdvancedHandler()

	points := generateHandlerTestPoints(20)
	reqBody := types.TSPPathRequest{
		RelicID:    1,
		Points:     points,
		Algorithm:  "two_opt",
		RobotSpeed: 50,
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/advanced/plan-tsp-path", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var result types.TSPPathResult
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(result.OrderedPoints) != len(points) {
		t.Errorf("expected %d points, got %d", len(points), len(result.OrderedPoints))
	}
	if result.TotalDistance <= 0 {
		t.Error("expected positive distance")
	}
	t.Logf("TSP handler: distance=%.2f, time=%.1fs, iterations=%d",
		result.TotalDistance, result.TotalTimeSeconds, result.Iterations)
}

func TestPlanTSPPath_EmptyPoints(t *testing.T) {
	_, r := setupAdvancedHandler()

	reqBody := types.TSPPathRequest{
		RelicID: 1,
		Points:  []types.CleaningPoint{},
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/advanced/plan-tsp-path", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestPlanTSPPath_InvalidJSON(t *testing.T) {
	_, r := setupAdvancedHandler()

	req, _ := http.NewRequest("POST", "/api/v1/advanced/plan-tsp-path",
		bytes.NewBufferString("not valid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for invalid JSON, got %d", w.Code)
	}
}

func TestPredictRoughness_Handler(t *testing.T) {
	_, r := setupAdvancedHandler()

	reqBody := types.RoughnessPredictionRequest{
		RelicID:          1,
		EnergyDensity:    1.8,
		LaserPower:       200,
		PulseDuration:    1000,
		ScanSpeed:        80,
		InitialRoughness: 30,
		OverlapRate:      0.5,
		MineralComposition: map[string]float32{
			"calcium_sulfate": 0.55,
			"calcite":         0.25,
			"dolomite":        0.12,
			"silicate":        0.08,
		},
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/advanced/predict-roughness", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var result types.RoughnessPredictionResult
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if result.PredictedRoughness <= 0 {
		t.Error("predicted roughness should be positive")
	}
	if result.Confidence <= 0 || result.Confidence > 1 {
		t.Errorf("confidence should be in (0,1], got %f", result.Confidence)
	}
	t.Logf("Roughness handler: predicted=%.2f μm, risk=%s, confidence=%.0f%%",
		result.PredictedRoughness, result.RiskLevel, result.Confidence*100)
}

func TestPredictRoughness_DefaultMinerals(t *testing.T) {
	_, r := setupAdvancedHandler()

	reqBody := types.RoughnessPredictionRequest{
		RelicID:          1,
		EnergyDensity:    1.5,
		LaserPower:       150,
		PulseDuration:    800,
		ScanSpeed:        100,
		InitialRoughness: 25,
		OverlapRate:      0.4,
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/advanced/predict-roughness", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var result types.RoughnessPredictionResult
	json.Unmarshal(w.Body.Bytes(), &result)

	t.Logf("Default minerals: predicted Ra=%.2f μm", result.PredictedRoughness)
}

func TestPredictRescaling_Handler(t *testing.T) {
	_, r := setupAdvancedHandler()

	history := make([]float32, 30)
	val := float32(0.02)
	for i := 0; i < 30; i++ {
		val += 0.005
		history[i] = val
	}

	reqBody := types.RescalingPredictionRequest{
		RelicID:          1,
		HistoryData:      history,
		Hours:            24,
		SO2Concentration: 25,
		Humidity:         65,
		Temperature:      16,
		PostCleaning:     false,
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/advanced/predict-rescaling", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var result types.RescalingPredictionResult
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(result.PredictedThickness) != 24 {
		t.Errorf("expected 24 predictions, got %d", len(result.PredictedThickness))
	}
	t.Logf("Rescaling handler: risk=%s, ARIMA=(%d,%d,%d), trigger=%v",
		result.RiskLevel, result.ARIMAParams[0], result.ARIMAParams[1], result.ARIMAParams[2],
		result.WarningTriggerHour)
}

func TestPredictRescaling_PostCleaning(t *testing.T) {
	_, r := setupAdvancedHandler()

	history := make([]float32, 20)
	val := float32(0.03)
	for i := 0; i < 20; i++ {
		val += 0.004
		history[i] = val
	}

	reqBody := types.RescalingPredictionRequest{
		RelicID:          1,
		HistoryData:      history,
		Hours:            48,
		SO2Concentration: 50,
		Humidity:         80,
		Temperature:      28,
		PostCleaning:     true,
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/advanced/predict-rescaling", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var result types.RescalingPredictionResult
	json.Unmarshal(w.Body.Bytes(), &result)
	t.Logf("Post-cleaning high-pollution scenario: risk=%s, final=%.4f mm",
		result.RiskLevel, result.PredictedThickness[len(result.PredictedThickness)-1])
}

func TestSimulateRobot_Handler(t *testing.T) {
	_, r := setupAdvancedHandler()

	path := generateHandlerTestPoints(8)
	reqBody := types.RobotSimulationRequest{
		RelicID:       1,
		Path:          path,
		StartPosition: [3]float32{0, 0, 0},
		SpeedFactor:   2.0,
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/advanced/simulate-robot", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var result types.RobotSimulationResult
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if result.TotalFrames == 0 {
		t.Error("expected frames")
	}
	if len(result.Frames) != result.TotalFrames {
		t.Errorf("frames len mismatch: %d vs %d", len(result.Frames), result.TotalFrames)
	}
	t.Logf("Robot sim handler: %d frames, %.2f sec, area=%.2f mm²",
		result.TotalFrames, result.DurationSec, result.AreaCleaned)
}

func TestSimulateRobot_EmptyPath(t *testing.T) {
	_, r := setupAdvancedHandler()

	reqBody := types.RobotSimulationRequest{
		RelicID:       1,
		Path:          []types.CleaningPoint{},
		StartPosition: [3]float32{0, 0, 0},
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/advanced/simulate-robot", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestAllAdvancedEndpoints(t *testing.T) {
	testCases := []struct {
		name   string
		method string
		path   string
		body   interface{}
		check  func(*testing.T, int, []byte)
	}{
		{
			"PlanTSP",
			"POST",
			"/api/v1/advanced/plan-tsp-path",
			types.TSPPathRequest{
				RelicID: 1,
				Points:  generateHandlerTestPoints(15),
			},
			func(t *testing.T, code int, body []byte) {
				if code != 200 {
					t.Errorf("expected 200, got %d", code)
				}
			},
		},
		{
			"PredictRoughness",
			"POST",
			"/api/v1/advanced/predict-roughness",
			types.RoughnessPredictionRequest{
				RelicID:          1,
				EnergyDensity:    2.0,
				LaserPower:       180,
				PulseDuration:    900,
				ScanSpeed:        90,
				InitialRoughness: 28,
				OverlapRate:      0.45,
			},
			func(t *testing.T, code int, body []byte) {
				if code != 200 {
					t.Errorf("expected 200, got %d", code)
				}
			},
		},
		{
			"PredictRescaling",
			"POST",
			"/api/v1/advanced/predict-rescaling",
			types.RescalingPredictionRequest{
				RelicID:          1,
				Hours:            12,
				SO2Concentration: 30,
				Humidity:         70,
				Temperature:      20,
			},
			func(t *testing.T, code int, body []byte) {
				if code != 200 {
					t.Errorf("expected 200, got %d", code)
				}
			},
		},
		{
			"SimulateRobot",
			"POST",
			"/api/v1/advanced/simulate-robot",
			types.RobotSimulationRequest{
				RelicID:       1,
				Path:          generateHandlerTestPoints(5),
				StartPosition: [3]float32{0, 0, 0},
			},
			func(t *testing.T, code int, body []byte) {
				if code != 200 {
					t.Errorf("expected 200, got %d", code)
				}
			},
		},
	}

	_, r := setupAdvancedHandler()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bodyBytes, _ := json.Marshal(tc.body)
			req, _ := http.NewRequest(tc.method, tc.path, bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			tc.check(t, w.Code, w.Body.Bytes())
		})
	}
}

func TestAdvancedHandlerRegisterRoutes(t *testing.T) {
	handler, r := setupAdvancedHandler()

	apiGroup := r.Group("/api/v1")
	handler.RegisterRoutes(apiGroup)

	routes := []string{
		"POST /api/v1/advanced/plan-tsp-path",
		"POST /api/v1/advanced/predict-roughness",
		"POST /api/v1/advanced/predict-rescaling",
		"POST /api/v1/advanced/simulate-robot",
	}

	for _, route := range routes {
		t.Logf("Registered route: %s", route)
	}

	if len(r.Routes()) == 0 {
		t.Error("no routes registered")
	}
}

func TestTSPAlgorithmVariants(t *testing.T) {
	_, r := setupAdvancedHandler()
	points := generateHandlerTestPoints(25)

	algorithms := []string{"nearest", "two_opt", "priority", "random"}

	for _, alg := range algorithms {
		t.Run("alg_"+alg, func(t *testing.T) {
			reqBody := types.TSPPathRequest{
				RelicID:    1,
				Points:     points,
				Algorithm:  alg,
				RobotSpeed: 60,
			}
			body, _ := json.Marshal(reqBody)

			req, _ := http.NewRequest("POST", "/api/v1/advanced/plan-tsp-path", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("algorithm %s: expected 200, got %d", alg, w.Code)
			}

			var result types.TSPPathResult
			json.Unmarshal(w.Body.Bytes(), &result)
			t.Logf("Algorithm %s: distance=%.2f mm, time=%.1f s",
				alg, result.TotalDistance, result.TotalTimeSeconds)
		})
	}
}

func TestBoundaryConditions(t *testing.T) {
	_, r := setupAdvancedHandler()

	t.Run("Single point TSP", func(t *testing.T) {
		reqBody := types.TSPPathRequest{
			RelicID: 1,
			Points:  []types.CleaningPoint{{ID: 0, X: 0, Y: 0, Z: 0}},
		}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/v1/advanced/plan-tsp-path", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		var result types.TSPPathResult
		json.Unmarshal(w.Body.Bytes(), &result)
		if len(result.OrderedPoints) != 1 {
			t.Errorf("expected 1 point, got %d", len(result.OrderedPoints))
		}
		if result.TotalDistance != 0 {
			t.Errorf("expected 0 distance, got %f", result.TotalDistance)
		}
	})

	t.Run("Short history ARIMA", func(t *testing.T) {
		reqBody := types.RescalingPredictionRequest{
			RelicID:     1,
			HistoryData: []float32{0.01, 0.02},
			Hours:       6,
		}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/v1/advanced/predict-rescaling", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		var result types.RescalingPredictionResult
		json.Unmarshal(w.Body.Bytes(), &result)
		if len(result.PredictedThickness) != 6 {
			t.Errorf("expected 6 predictions, got %d", len(result.PredictedThickness))
		}
	})

	t.Run("Extreme energy roughness", func(t *testing.T) {
		reqBody := types.RoughnessPredictionRequest{
			RelicID:          1,
			EnergyDensity:    10.0,
			LaserPower:       500,
			PulseDuration:    5000,
			ScanSpeed:        5,
			InitialRoughness: 100,
			OverlapRate:      0.9,
		}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/v1/advanced/predict-roughness", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		var result types.RoughnessPredictionResult
		json.Unmarshal(w.Body.Bytes(), &result)
		t.Logf("Extreme energy: Ra=%.2f μm, risk=%s", result.PredictedRoughness, result.RiskLevel)
	})
}
