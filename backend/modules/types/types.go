package types

type CleaningPoint struct {
	ID        int     `json:"id"`
	X         float32 `json:"x"`
	Y         float32 `json:"y"`
	Z         float32 `json:"z"`
	Thickness float32 `json:"thickness"`
	Area      float32 `json:"area"`
	Priority  int     `json:"priority"`
}

type TSPPathRequest struct {
	RelicID    uint64          `json:"relic_id"`
	Points     []CleaningPoint `json:"points"`
	StartPoint *CleaningPoint  `json:"start_point,omitempty"`
	RobotSpeed float32         `json:"robot_speed"`
	Algorithm  string          `json:"algorithm"`
}

type TSPPathResult struct {
	RelicID          uint64          `json:"relic_id"`
	OrderedPoints    []CleaningPoint `json:"ordered_points"`
	TotalDistance    float32         `json:"total_distance"`
	TotalTimeSeconds float32         `json:"total_time_seconds"`
	PathIndices      []int           `json:"path_indices"`
	Algorithm        string          `json:"algorithm"`
	Iterations       int             `json:"iterations"`
}

type RoughnessPredictionRequest struct {
	RelicID            uint64             `json:"relic_id"`
	EnergyDensity      float32            `json:"energy_density"`
	LaserPower         float32            `json:"laser_power"`
	PulseDuration      float32            `json:"pulse_duration"`
	ScanSpeed          float32            `json:"scan_speed"`
	MineralComposition map[string]float32 `json:"mineral_composition"`
	InitialRoughness   float32            `json:"initial_roughness"`
	OverlapRate        float32            `json:"overlap_rate"`
}

type RoughnessPredictionResult struct {
	RelicID            uint64             `json:"relic_id"`
	PredictedRoughness float32            `json:"predicted_roughness"`
	Confidence         float32            `json:"confidence"`
	FeatureImportance  map[string]float32 `json:"feature_importance"`
	RoughnessRange     [2]float32         `json:"roughness_range"`
	RiskLevel          string             `json:"risk_level"`
}

type RescalingPredictionRequest struct {
	RelicID          uint64    `json:"relic_id"`
	HistoryData      []float32 `json:"history_data"`
	Hours            int       `json:"hours"`
	SO2Concentration float32   `json:"so2_concentration"`
	Humidity         float32   `json:"humidity"`
	Temperature      float32   `json:"temperature"`
	PostCleaning     bool      `json:"post_cleaning"`
}

type RescalingPredictionResult struct {
	RelicID            uint64    `json:"relic_id"`
	PredictedRates     []float32 `json:"predicted_rates"`
	PredictedThickness []float32 `json:"predicted_thickness"`
	Hours              []int     `json:"hours"`
	RiskLevel          string    `json:"risk_level"`
	WarningThreshold   float32   `json:"warning_threshold"`
	WarningTriggerHour *int      `json:"warning_trigger_hour,omitempty"`
	ARIMAParams        [3]int    `json:"arima_params"`
	Confidence         float32   `json:"confidence"`
}

type LaserCleaningResult struct {
	RelicID                uint64  `json:"relic_id"`
	OptimalPower           float32 `json:"optimal_power"`
	OptimalPulse           float32 `json:"optimal_pulse"`
	OptimalSpeed           float32 `json:"optimal_speed"`
	PredictedDepth         float32 `json:"predicted_depth"`
	PredictedEnergyDensity float32 `json:"predicted_energy_density"`
	AblationThreshold      float32 `json:"ablation_threshold"`
	Confidence             float32 `json:"confidence"`
	SafetyWarning          string  `json:"safety_warning"`
}

type RobotSimulationRequest struct {
	RelicID       uint64               `json:"relic_id"`
	Path          []CleaningPoint      `json:"path"`
	StartPosition [3]float32           `json:"start_position"`
	LaserParams   LaserCleaningResult  `json:"laser_params"`
	SpeedFactor   float32              `json:"speed_factor"`
}

type RobotFrame struct {
	Timestamp      int64       `json:"timestamp"`
	RobotPosition  [3]float32  `json:"robot_position"`
	RobotRotation  [3]float32  `json:"robot_rotation"`
	CurrentPointID int         `json:"current_point_id"`
	LaserActive    bool        `json:"laser_active"`
	CleaningArea   [][]float32 `json:"cleaning_area"`
	Progress       float32     `json:"progress"`
}

type RobotSimulationResult struct {
	RelicID     uint64       `json:"relic_id"`
	Frames      []RobotFrame `json:"frames"`
	TotalFrames int          `json:"total_frames"`
	DurationSec float32      `json:"duration_sec"`
	AreaCleaned float32      `json:"area_cleaned"`
}
