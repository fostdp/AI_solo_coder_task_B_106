-- ============================================================
-- ClickHouse 索引优化方案：ORDER BY 排序键重排
-- 原问题：ORDER BY (relic_id, sensor_id, timestamp) 
--           → 按时间范围查询时，稀疏索引跳数过多，查询慢
-- 修复后：  ORDER BY (timestamp, relic_id, sensor_id)
--           → 时间维度为主键首列，时间范围查询可直接定位数据范围
-- ============================================================

USE stone_relic;

-- ============================================================
-- 1. sensor_data 表：时间首列排序优化
-- ============================================================

-- 方案A：直接重建（推荐，数据量大时使用 OPTIMIZE 重构
OPTIMIZE TABLE sensor_data FINAL;

-- 方案B：新建优化后的表并迁移数据（生产环境用
CREATE TABLE IF NOT EXISTS sensor_data_opt (
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
ORDER BY (timestamp, relic_id, sensor_id)
TTL timestamp + INTERVAL 1 YEAR
SETTINGS 
  index_granularity = 8192,
  min_bytes_for_wide_part = '10M',
  min_bytes_for_compact_part = '1M';

-- 数据迁移语句（生产环境分批执行）
-- INSERT INTO sensor_data_opt SELECT * FROM sensor_data;

-- 切换表名
-- RENAME TABLE sensor_data TO sensor_data_old;
-- RENAME TABLE sensor_data_opt TO sensor_data;

-- ============================================================
-- 2. 跳数索引（Skip Index）：针对高基数列
-- ============================================================

-- 针对 sensor_id 的 minmax 跳数索引（已存在于排序键中，额外增强）
ALTER TABLE sensor_data 
  ADD INDEX IF NOT EXISTS idx_sensor_id_minmax sensor_id TYPE minmax GRANULARITY 1;

-- 针对 relic_id 的 set 跳数索引
ALTER TABLE sensor_data 
  ADD INDEX IF NOT EXISTS idx_relic_id_set relic_id TYPE set(100) GRANULARITY 4;

-- 针对 value 的布隆过滤器索引（快速过滤阈值告警查询）
ALTER TABLE sensor_data 
  ADD INDEX IF NOT EXISTS idx_value_bloom value TYPE bloom_filter GRANULARITY 4;

-- 物化索引到优化后表
-- ALTER TABLE sensor_data_opt 
--   ADD INDEX IF NOT EXISTS idx_sensor_id_minmax sensor_id TYPE minmax GRANULARITY 1;
-- ALTER TABLE sensor_data_opt 
--   ADD INDEX IF NOT EXISTS idx_relic_id_set relic_id TYPE set(100) GRANULARITY 4;
-- ALTER TABLE sensor_data_opt 
--   ADD INDEX IF NOT EXISTS idx_value_bloom value TYPE bloom_filter GRANULARITY 4;

-- ============================================================
-- 3. 告警记录表优化
-- ============================================================

-- 告警记录表按时间排序优化
CREATE TABLE IF NOT EXISTS alert_record_opt (
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
ORDER BY (created_at, relic_id, level)
SETTINGS index_granularity = 8192;

-- ============================================================
-- 4. 清洗记录表优化
-- ============================================================

CREATE TABLE IF NOT EXISTS cleaning_record_opt (
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
ORDER BY (created_at, relic_id, area_id)
SETTINGS index_granularity = 8192;

-- ============================================================
-- 5. 参数优化日志表优化
-- ============================================================

CREATE TABLE IF NOT EXISTS cleaning_parameter_opt_log_opt (
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
ORDER BY (created_at, relic_id)
SETTINGS index_granularity = 8192;

-- ============================================================
-- 6. 视图适配（适配新排序键
-- ============================================================

-- 最新传感器数据视图（优化版本）
DROP VIEW IF EXISTS v_latest_sensor_data_opt
AS
SELECT
    sensor_id,
    relic_id,
    argMax(timestamp, timestamp) AS latest_time,
    argMax(value, timestamp) AS latest_value,
    argMax(unit, timestamp) AS latest_unit,
    argMax(so2_concentration, timestamp) AS latest_so2,
    argMax(humidity, timestamp) AS latest_humidity,
    argMax(temperature, timestamp) AS latest_temperature
FROM sensor_data
GROUP BY relic_id, sensor_id;

-- 日统计视图（优化版本）
DROP VIEW IF EXISTS v_daily_statistics_opt
AS
SELECT
    relic_id,
    toDate(timestamp) AS date,
    avgIf(value, sensor_id IN (
        SELECT id FROM sensor WHERE type = 'ultrasonic'
    )) AS avg_thickness,
    maxIf(value, sensor_id IN (
        SELECT id FROM sensor WHERE type = 'ultrasonic'
    )) AS max_thickness,
    avgIf(value, sensor_id IN (
        SELECT id FROM sensor WHERE type = 'roughness'
    )) AS avg_roughness,
    maxIf(value, sensor_id IN (
        SELECT id FROM sensor WHERE type = 'roughness'
    )) AS max_roughness,
    avg(so2_concentration) AS avg_so2,
    avg(humidity) AS avg_humidity,
    avg(temperature) AS avg_temperature,
    count() AS data_count
FROM sensor_data
GROUP BY relic_id, toDate(timestamp);

-- ============================================================
-- 7. 查询性能对比验证 SQL
-- ============================================================

-- 查询最近24小时数据，按文物ID 查询性能对比
-- 优化前（relic_id在前）：
-- SELECT count() FROM sensor_data WHERE relic_id = 1 AND timestamp > now() - INTERVAL 1 DAY;
-- 优化后（timestamp在前）：
-- SELECT count() FROM sensor_data_opt WHERE relic_id = 1 AND timestamp > now() - INTERVAL 1 DAY;

-- 按时间范围查询（跨度查询性能对比
-- SELECT max(value) FROM sensor_data WHERE timestamp BETWEEN '2025-01-01' AND '2025-01-07';

-- ============================================================
-- 8. 合并树参数调优建议
-- ============================================================

-- 调大索引粒度（数据量大时）
-- ALTER TABLE sensor_data MODIFY SETTING index_granularity = 16384;

-- 启用自适应粒度自适应
-- ALTER TABLE sensor_data MODIFY SETTING index_granularity = 8192;
-- ALTER TABLE sensor_data MODIFY SETTING min_index_granularity_bytes = 1048576;

-- 启用主键压缩
-- ALTER TABLE sensor_data MODIFY SETTING ratio_of_defaults_for_sparse_serialization = 0.5;

-- ============================================================
-- 9. 批量迁移脚本（一键执行）
-- ============================================================
-- 执行步骤：
-- 1. 执行本文件建表
-- 2. 执行数据迁移（INSERT ... SELECT
-- 3. 验证数据一致性
-- 4. RENAME 切换表名
-- 5. 重建视图指向新表
-- ============================================================
