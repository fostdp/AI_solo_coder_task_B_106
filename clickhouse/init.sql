CREATE DATABASE IF NOT EXISTS stone_relic;

USE stone_relic;

CREATE TABLE IF NOT EXISTS stone_relic (
  id UInt64,
  name String,
  location String,
  model_path String,
  created_at DateTime DEFAULT now()
) ENGINE = MergeTree()
ORDER BY id;

CREATE TABLE IF NOT EXISTS sensor (
  id UInt64,
  relic_id UInt64,
  type Enum8('ultrasonic' = 1, 'roughness' = 2),
  model String,
  position_x Float32,
  position_y Float32,
  created_at DateTime DEFAULT now()
) ENGINE = MergeTree()
ORDER BY id;

CREATE TABLE IF NOT EXISTS sensor_data (
  id UInt64,
  sensor_id UInt64,
  relic_id UInt64,
  timestamp DateTime,
  value Float32,
  unit LowCardinality(String),
  so2_concentration Float32,
  humidity Float32,
  temperature Float32
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (relic_id, sensor_id, timestamp)
TTL timestamp + INTERVAL 1 YEAR
SETTINGS index_granularity = 8192;

CREATE TABLE IF NOT EXISTS alert_record (
  id UInt64,
  relic_id UInt64,
  sensor_id UInt64,
  type Enum8('thickness' = 1, 'roughness' = 2),
  level Enum8('warning' = 1, 'critical' = 2),
  value Float32,
  threshold Float32,
  message String,
  created_at DateTime DEFAULT now()
) ENGINE = MergeTree()
ORDER BY (relic_id, created_at);

CREATE TABLE IF NOT EXISTS cleaning_record (
  id UInt64,
  relic_id UInt64,
  area_id UInt32,
  laser_power Float32,
  pulse_duration Float32,
  scan_speed Float32,
  predicted_depth Float32,
  actual_depth Float32,
  operator String,
  created_at DateTime DEFAULT now()
) ENGINE = MergeTree()
ORDER BY (relic_id, created_at);

CREATE TABLE IF NOT EXISTS cleaning_parameter_opt_log (
  id UInt64,
  relic_id UInt64,
  area_id UInt32,
  target_thickness Float32,
  material_type String,
  optimal_power Float32,
  optimal_pulse Float32,
  optimal_speed Float32,
  predicted_energy_density Float32,
  ablation_threshold Float32,
  confidence Float32,
  created_at DateTime DEFAULT now()
) ENGINE = MergeTree()
ORDER BY (relic_id, created_at);

INSERT INTO stone_relic (id, name, location, model_path) VALUES
(1, '云冈石窟-第20窟大佛', '山西大同', '/models/yungang_20.glb'),
(2, '乐山大佛', '四川乐山', '/models/leshan.glb'),
(3, '龙门石窟-卢舍那大佛', '河南洛阳', '/models/longmen_lushena.glb'),
(4, '敦煌莫高窟-第96窟', '甘肃敦煌', '/models/dunhuang_96.glb'),
(5, '麦积山石窟-第44窟', '甘肃天水', '/models/maijishan_44.glb'),
(6, '大足石刻-宝顶山', '重庆大足', '/models/dazu_baodingshan.glb'),
(7, '响堂山石窟-北响堂', '河北邯郸', '/models/xiangtangshan_north.glb'),
(8, '天龙山石窟', '山西太原', '/models/tianlongshan.glb'),
(9, '巩义石窟寺', '河南巩义', '/models/gongyi.glb'),
(10, '须弥山石窟', '宁夏固原', '/models/xumishan.glb');

INSERT INTO sensor (id, relic_id, type, model, position_x, position_y)
SELECT
  (r.id - 1) * 5 + n + 1,
  r.id,
  'ultrasonic',
  'US-300',
  n * 0.2,
  0.5
FROM stone_relic r
ARRAY JOIN range(3) AS n;

INSERT INTO sensor (id, relic_id, type, model, position_x, position_y)
SELECT
  100 + (r.id - 1) * 3 + n + 1,
  r.id,
  'roughness',
  'RT-200',
  n * 0.3 + 0.1,
  0.8
FROM stone_relic r
ARRAY JOIN range(2) AS n;

CREATE VIEW IF NOT EXISTS v_latest_sensor_data
ENGINE = MergeTree()
ORDER BY (relic_id, sensor_id)
AS
SELECT
    relic_id,
    sensor_id,
    argMax(timestamp, timestamp) AS latest_time,
    argMax(value, timestamp) AS latest_value,
    argMax(unit, timestamp) AS latest_unit,
    argMax(so2_concentration, timestamp) AS latest_so2,
    argMax(humidity, timestamp) AS latest_humidity,
    argMax(temperature, timestamp) AS latest_temperature
FROM sensor_data
GROUP BY relic_id, sensor_id;

CREATE VIEW IF NOT EXISTS v_daily_statistics
ENGINE = MergeTree()
ORDER BY (relic_id, toDate(timestamp))
AS
SELECT
    relic_id,
    toDate(timestamp) AS date,
    avgIf(value, sensor_id IN (SELECT id FROM sensor WHERE type = 'ultrasonic')) AS avg_thickness,
    maxIf(value, sensor_id IN (SELECT id FROM sensor WHERE type = 'ultrasonic')) AS max_thickness,
    avgIf(value, sensor_id IN (SELECT id FROM sensor WHERE type = 'roughness')) AS avg_roughness,
    maxIf(value, sensor_id IN (SELECT id FROM sensor WHERE type = 'roughness')) AS max_roughness,
    avg(so2_concentration) AS avg_so2,
    avg(humidity) AS avg_humidity,
    avg(temperature) AS avg_temperature,
    count() AS data_count
FROM sensor_data
GROUP BY relic_id, toDate(timestamp);
