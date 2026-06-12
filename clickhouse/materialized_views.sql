USE stone_relic;

CREATE TABLE IF NOT EXISTS mv_daily_scale_growth_rate
(
    relic_id UInt64,
    date Date,
    sensor_id UInt64,
    avg_thickness Float32,
    max_thickness Float32,
    min_thickness Float32,
    growth_rate_24h Float32,
    growth_rate_7d Float32,
    growth_rate_30d Float32,
    so2_avg Float32,
    humidity_avg Float32,
    temperature_avg Float32,
    data_points UInt32
)
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (relic_id, date, sensor_id)
TTL date + INTERVAL 1 YEAR;

CREATE MATERIALIZED VIEW IF NOT EXISTS mv_daily_scale_growth_rate_view
TO mv_daily_scale_growth_rate
AS
SELECT
    relic_id,
    toDate(timestamp) AS date,
    sensor_id,
    avgIf(value, unit = 'mm') AS avg_thickness,
    maxIf(value, unit = 'mm') AS max_thickness,
    minIf(value, unit = 'mm') AS min_thickness,
    0 AS growth_rate_24h,
    0 AS growth_rate_7d,
    0 AS growth_rate_30d,
    avg(so2_concentration) AS so2_avg,
    avg(humidity) AS humidity_avg,
    avg(temperature) AS temperature_avg,
    count() AS data_points
FROM sensor_data
WHERE unit = 'mm'
GROUP BY relic_id, toDate(timestamp), sensor_id;

CREATE TABLE IF NOT EXISTS mv_hourly_sensor_stats
(
    relic_id UInt64,
    sensor_id UInt64,
    hour DateTime,
    unit LowCardinality(String),
    avg_value Float32,
    max_value Float32,
    min_value Float32,
    stddev_value Float32,
    data_points UInt32
)
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(hour)
ORDER BY (relic_id, sensor_id, hour)
TTL hour + INTERVAL 90 DAY;

CREATE MATERIALIZED VIEW IF NOT EXISTS mv_hourly_sensor_stats_view
TO mv_hourly_sensor_stats
AS
SELECT
    relic_id,
    sensor_id,
    toStartOfHour(timestamp) AS hour,
    unit,
    avg(value) AS avg_value,
    max(value) AS max_value,
    min(value) AS min_value,
    stddevSamp(value) AS stddev_value,
    count() AS data_points
FROM sensor_data
GROUP BY relic_id, sensor_id, toStartOfHour(timestamp), unit;

CREATE TABLE IF NOT EXISTS mv_alert_summary
(
    relic_id UInt64,
    date Date,
    alert_type LowCardinality(String),
    severity LowCardinality(String),
    total_alerts UInt32,
    unique_sensors UInt32,
    avg_value Float32,
    max_value Float32,
    resolved_count UInt32
)
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (relic_id, date, alert_type, severity)
TTL date + INTERVAL 1 YEAR;

CREATE MATERIALIZED VIEW IF NOT EXISTS mv_alert_summary_view
TO mv_alert_summary
AS
SELECT
    relic_id,
    toDate(timestamp) AS date,
    alert_type,
    severity,
    count() AS total_alerts,
    uniqExact(sensor_id) AS unique_sensors,
    avg(value) AS avg_value,
    max(value) AS max_value,
    countIf(resolved) AS resolved_count
FROM alert_record
GROUP BY relic_id, toDate(timestamp), alert_type, severity;

CREATE TABLE IF NOT EXISTS mv_cleaning_summary
(
    relic_id UInt64,
    date Date,
    total_cleanings UInt32,
    avg_laser_power Float32,
    avg_pulse_duration Float32,
    avg_scan_speed Float32,
    avg_target_depth Float32,
    avg_actual_depth Float32,
    avg_effectiveness Float32,
    total_depth_removed Float32
)
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (relic_id, date)
TTL date + INTERVAL 2 YEAR;

CREATE MATERIALIZED VIEW IF NOT EXISTS mv_cleaning_summary_view
TO mv_cleaning_summary
AS
SELECT
    relic_id,
    toDate(timestamp) AS date,
    count() AS total_cleanings,
    avg(laser_power) AS avg_laser_power,
    avg(pulse_duration) AS avg_pulse_duration,
    avg(scan_speed) AS avg_scan_speed,
    avg(target_depth) AS avg_target_depth,
    avg(actual_depth) AS avg_actual_depth,
    avg(effectiveness) AS avg_effectiveness,
    sum(actual_depth) AS total_depth_removed
FROM cleaning_record
GROUP BY relic_id, toDate(timestamp);

CREATE VIEW IF NOT EXISTS v_scale_growth_trend AS
WITH daily_stats AS (
    SELECT
        relic_id,
        sensor_id,
        toDate(timestamp) AS date,
        avg(value) AS daily_avg
    FROM sensor_data
    WHERE unit = 'mm'
    GROUP BY relic_id, sensor_id, toDate(timestamp)
)
SELECT
    relic_id,
    sensor_id,
    date,
    daily_avg AS current_avg,
    daily_avg - anyValue(daily_avg) OVER (
        PARTITION BY relic_id, sensor_id
        ORDER BY date
        ROWS BETWEEN 1 PRECEDING AND 1 PRECEDING
    ) AS growth_day_over_day,
    daily_avg - anyValue(daily_avg) OVER (
        PARTITION BY relic_id, sensor_id
        ORDER BY date
        ROWS BETWEEN 7 PRECEDING AND 7 PRECEDING
    ) AS growth_7day,
    avg(daily_avg) OVER (
        PARTITION BY relic_id, sensor_id
        ORDER BY date
        ROWS BETWEEN 7 PRECEDING AND CURRENT ROW
    ) AS moving_avg_7d,
    avg(daily_avg) OVER (
        PARTITION BY relic_id, sensor_id
        ORDER BY date
        ROWS BETWEEN 30 PRECEDING AND CURRENT ROW
    ) AS moving_avg_30d
FROM daily_stats
ORDER BY relic_id, sensor_id, date DESC;

CREATE VIEW IF NOT EXISTS v_environmental_correlation AS
SELECT
    relic_id,
    toStartOfDay(timestamp) AS date,
    corr(value, so2_concentration) AS corr_so2_thickness,
    corr(value, humidity) AS corr_humidity_thickness,
    corr(value, temperature) AS corr_temp_thickness,
    avg(so2_concentration) AS avg_so2,
    avg(humidity) AS avg_humidity,
    avg(temperature) AS avg_temp,
    avgIf(value, unit = 'mm') AS avg_thickness
FROM sensor_data
WHERE unit = 'mm'
GROUP BY relic_id, toStartOfDay(timestamp)
ORDER BY relic_id, date DESC;

SET allow_experimental_analyzer = 1;
