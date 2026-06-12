package models

import "time"

type StoneRelic struct {
	ID        uint64    `json:"id"`
	Name      string    `json:"name"`
	Location  string    `json:"location"`
	ModelPath string    `json:"model_path"`
	CreatedAt time.Time `json:"created_at"`
}

type Sensor struct {
	ID        uint64    `json:"id"`
	RelicID   uint64    `json:"relic_id"`
	Type      string    `json:"type"`
	Model     string    `json:"model"`
	PositionX float32   `json:"position_x"`
	PositionY float32   `json:"position_y"`
	CreatedAt time.Time `json:"created_at"`
}

type SensorData struct {
	ID                uint64    `json:"id"`
	SensorID          uint64    `json:"sensor_id"`
	RelicID           uint64    `json:"relic_id"`
	Timestamp         time.Time `json:"timestamp"`
	Value             float32   `json:"value"`
	Unit              string    `json:"unit"`
	SO2Concentration  float32   `json:"so2_concentration"`
	Humidity          float32   `json:"humidity"`
	Temperature       float32   `json:"temperature"`
}

type SensorDataBatch struct {
	Data []SensorData `json:"data"`
}

type LatestSensorData struct {
	RelicID          uint64    `json:"relic_id"`
	SensorID         uint64    `json:"sensor_id"`
	LatestTime       time.Time `json:"latest_time"`
	LatestValue      float32   `json:"latest_value"`
	LatestUnit       string    `json:"latest_unit"`
	LatestSO2        float32   `json:"latest_so2"`
	LatestHumidity   float32   `json:"latest_humidity"`
	LatestTemperature float32  `json:"latest_temperature"`
}

type AlertRecord struct {
	ID              uint64     `json:"id"`
	RelicID         uint64     `json:"relic_id"`
	SensorID        uint64     `json:"sensor_id"`
	Timestamp       time.Time  `json:"timestamp"`
	AlertType       string     `json:"alert_type"`
	Severity        string     `json:"severity"`
	Message         string     `json:"message"`
	Value           float32    `json:"value"`
	Threshold       float32    `json:"threshold"`
	Resolved        bool       `json:"resolved"`
	ResolvedAt      *time.Time `json:"resolved_at,omitempty"`
	ResolutionNotes string     `json:"resolution_notes,omitempty"`
}

type CleaningRecord struct {
	ID             uint64    `json:"id"`
	RelicID        uint64    `json:"relic_id"`
	AreaID         uint32    `json:"area_id"`
	Timestamp      time.Time `json:"timestamp"`
	LaserPower     float32   `json:"laser_power"`
	PulseDuration  float32   `json:"pulse_duration"`
	ScanSpeed      float32   `json:"scan_speed"`
	TargetDepth    float32   `json:"target_depth"`
	ActualDepth    float32   `json:"actual_depth"`
	EnergyDensity  float32   `json:"energy_density"`
	Effectiveness  float32   `json:"effectiveness"`
	OperatorNotes  string    `json:"operator_notes,omitempty"`
}

type CleaningParameterOpt struct {
	ID                      uint64    `json:"id"`
	RelicID                 uint64    `json:"relic_id"`
	AreaID                  uint32    `json:"area_id"`
	TargetThickness         float32   `json:"target_thickness"`
	MaterialType            string    `json:"material_type"`
	OptimalPower            float32   `json:"optimal_power"`
	OptimalPulse            float32   `json:"optimal_pulse"`
	OptimalSpeed            float32   `json:"optimal_speed"`
	PredictedEnergyDensity  float32   `json:"predicted_energy_density"`
	AblationThreshold       float32   `json:"ablation_threshold"`
	Confidence              float32   `json:"confidence"`
	CreatedAt               time.Time `json:"created_at"`
}

type CleaningParameterOptLog struct {
	ID               uint64    `json:"id"`
	RelicID          uint64    `json:"relic_id"`
	Timestamp        time.Time `json:"timestamp"`
	RequestedPower   float32   `json:"requested_power"`
	RequestedPulse   float32   `json:"requested_pulse"`
	RequestedSpeed   float32   `json:"requested_speed"`
	OptimalPower     float32   `json:"optimal_power"`
	OptimalPulse     float32   `json:"optimal_pulse"`
	OptimalSpeed     float32   `json:"optimal_speed"`
	TargetDepth      float32   `json:"target_depth"`
	PredictedDepth   float32   `json:"predicted_depth"`
	OptimizationGain float32   `json:"optimization_gain"`
}

type ScaleGrowthPrediction struct {
	Hours               int       `json:"hours"`
	InitialThickness    float32   `json:"initial_thickness"`
	SO2Concentration    float32   `json:"so2_concentration"`
	Humidity            float32   `json:"humidity"`
	Temperature         float32   `json:"temperature"`
	PredictedThickness  []float32 `json:"predicted_thickness"`
	FinalThickness      float32   `json:"final_thickness"`
	GrowthRate          float32   `json:"growth_rate"`
	SaturationFactor    float32   `json:"saturation_factor"`
}

type LaserCleaningRequest struct {
	TargetThickness  float32 `json:"target_thickness"`
	MaterialType     string  `json:"material_type"`
	RelicID          uint64  `json:"relic_id"`
	AreaID           uint32  `json:"area_id"`
	SurfaceRoughness float32 `json:"surface_roughness"`
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

type DailyStatistics struct {
	RelicID       uint64    `json:"relic_id"`
	Date          time.Time `json:"date"`
	AvgThickness  float32   `json:"avg_thickness"`
	MaxThickness  float32   `json:"max_thickness"`
	AvgRoughness  float32   `json:"avg_roughness"`
	MaxRoughness  float32   `json:"max_roughness"`
	AvgSO2        float32   `json:"avg_so2"`
	AvgHumidity   float32   `json:"avg_humidity"`
	AvgTemperature float32  `json:"avg_temperature"`
	DataCount     uint64    `json:"data_count"`
}

type RelicDetail struct {
	StoneRelic
	Sensors        []Sensor            `json:"sensors"`
	LatestData     []LatestSensorData  `json:"latest_data"`
	MaxThickness   float32             `json:"max_thickness"`
	AvgRoughness   float32             `json:"avg_roughness"`
	AlertCount     int                 `json:"alert_count"`
}

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
	RelicID       uint64           `json:"relic_id"`
	Points        []CleaningPoint  `json:"points"`
	StartPoint    *CleaningPoint   `json:"start_point,omitempty"`
	RobotSpeed    float32          `json:"robot_speed"`
	Algorithm     string           `json:"algorithm"`
}

type TSPPathResult struct {
	RelicID          uint64           `json:"relic_id"`
	OrderedPoints    []CleaningPoint  `json:"ordered_points"`
	TotalDistance    float32          `json:"total_distance"`
	TotalTimeSeconds float32          `json:"total_time_seconds"`
	PathIndices      []int            `json:"path_indices"`
	Algorithm        string           `json:"algorithm"`
	Iterations       int              `json:"iterations"`
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
	RelicID          uint64             `json:"relic_id"`
	PredictedRoughness float32           `json:"predicted_roughness"`
	Confidence       float32            `json:"confidence"`
	FeatureImportance map[string]float32 `json:"feature_importance"`
	RoughnessRange   [2]float32         `json:"roughness_range"`
	RiskLevel        string             `json:"risk_level"`
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
	RelicID              uint64     `json:"relic_id"`
	PredictedRates       []float32  `json:"predicted_rates"`
	PredictedThickness   []float32  `json:"predicted_thickness"`
	Hours                []int      `json:"hours"`
	RiskLevel            string     `json:"risk_level"`
	WarningThreshold     float32    `json:"warning_threshold"`
	WarningTriggerHour   *int       `json:"warning_trigger_hour,omitempty"`
	ARIMAParams          [3]int     `json:"arima_params"`
	Confidence           float32    `json:"confidence"`
}

type RobotSimulationRequest struct {
	RelicID       uint64          `json:"relic_id"`
	Path          []CleaningPoint `json:"path"`
	StartPosition [3]float32      `json:"start_position"`
	LaserParams   LaserCleaningResult `json:"laser_params"`
	SpeedFactor   float32         `json:"speed_factor"`
}

type RobotFrame struct {
	Timestamp      int64      `json:"timestamp"`
	RobotPosition  [3]float32 `json:"robot_position"`
	RobotRotation  [3]float32 `json:"robot_rotation"`
	CurrentPointID int        `json:"current_point_id"`
	LaserActive    bool       `json:"laser_active"`
	CleaningArea   [][]float32 `json:"cleaning_area"`
	Progress       float32    `json:"progress"`
}

type RobotSimulationResult struct {
	RelicID      uint64       `json:"relic_id"`
	Frames       []RobotFrame `json:"frames"`
	TotalFrames  int          `json:"total_frames"`
	DurationSec  float32      `json:"duration_sec"`
	AreaCleaned  float32      `json:"area_cleaned"`

