# 古代石质文物表面结垢监测与激光清洗参数优化系统

## 系统概述

本系统针对云冈石窟、乐山大佛等10处石质文物，实现硫酸钙结垢厚度的实时监测、结垢生长动力学预测、激光清洗参数优化以及全流程的清洗效果模拟。

### 核心功能模块

| 模块 | 技术实现 | 说明 |
|------|----------|------|
| **数据采集层** | EtherCAT + UDP | 30台超声测厚仪 + 20台表面粗糙度仪，每2小时上报 |
| **后端服务** | Go + Gin + ClickHouse | 高性能时序数据存储与API服务 |
| **核心算法** | 经验模型 + 热传导方程 | 结垢生长动力学预测、激光烧蚀阈值计算 |
| **告警系统** | 钉钉机器人 + WebSocket | 厚度>3mm 或 粗糙度>50μm时触发推送 |
| **可视化层** | Three.js + Canvas | 3D石像模型、结垢等高线图、激光清洗粒子特效 |

---

## 目录结构

```
AI_solo_coder_task_A_055/
├── backend/                 # Go 后端服务
│   ├── main.go              # 入口文件
│   ├── config.yaml          # 系统配置
│   ├── go.mod
│   └── internal/
│       ├── config/          # 配置加载 (Viper)
│       ├── db/              # ClickHouse 连接池
│       ├── models/          # 数据模型定义
│       ├── router/          # Gin 路由注册
│       ├── handlers/        # HTTP 处理器
│       ├── services/        # 业务逻辑服务
│       ├── alert/           # 告警服务 + WebSocket Hub
│       └── algorithms/      # 核心算法实现
│
├── clickhouse/              # 数据库初始化
│   └── init.sql             # 表结构 + 视图 + 种子数据
│
├── simulator/               # EtherCAT 数据模拟器
│   ├── ethercat_sim.go      # 模拟器主程序
│   └── go.mod
│
├── frontend/                # Web 前端
│   ├── index.html           # 主页面
│   ├── css/
│   │   └── style.css        # 全局样式（深色仪表盘风格）
│   └── js/
│       ├── api.js           # REST API 封装
│       ├── ws.js            # WebSocket 客户端
│       ├── stats.js         # 数据统计渲染
│       ├── three-viewer.js  # Three.js 3D模型查看器
│       ├── contour.js       # Canvas 等高线渲染
│       ├── cleaning.js      # 激光清洗模拟 + 粒子特效
│       ├── trends.js        # 趋势图表
│       ├── algorithms.js    # 算法预测UI
│       └── app.js           # 主应用入口
│
└── README.md
```

---

## 快速启动

### 环境要求

- **Go** >= 1.21
- **ClickHouse** >= 22.0
- **现代浏览器**（支持WebGL 2.0：Chrome 80+ / Firefox 75+ / Edge 80+）

---

### 步骤1：启动 ClickHouse 并初始化

```bash
# Docker 方式启动 ClickHouse（推荐）
docker run -d --name stone-clickhouse \
  -p 8123:8123 -p 9000:9000 \
  -e CLICKHOUSE_DB=stone_relic \
  clickhouse/clickhouse-server:latest

# 执行初始化脚本
clickhouse-client --host 127.0.0.1 --port 9000 < clickhouse/init.sql
```

初始化脚本将创建：
- **stone_relic** 数据库
- 7张核心业务表（stone_relic, sensor, sensor_data, alert_record, cleaning_record, cleaning_parameter_opt_log）
- 2个物化视图（v_latest_sensor_data, v_daily_statistics）
- 自动注入10处文物 + 50个传感器的基础配置

---

### 步骤2：启动 Go 后端

```bash
cd backend

# 安装依赖
go mod download

# 配置文件已包含默认值，如需修改钉钉Webhook请编辑 config.yaml
# 启动服务
go run main.go
```

服务将在 `http://127.0.0.1:8080` 启动，健康检查：
```bash
curl http://127.0.0.1:8080/health
```

---

### 步骤3：启动 EtherCAT 数据模拟器

```bash
cd simulator
go mod download

# 启动模拟器（会自动回填1个月历史数据 + 实时模拟每10秒上报）
go run ethercat_sim.go
```

模拟器特性：
- 启动后回填 **过去1个月**（30天×12次/天=360批）的历史数据
- 按每10秒间隔持续模拟50台传感器上报
- 包含日周期、季周期、随机扰动、偶发异常值
- 按文物所在地差异化生成 SO₂/湿度/温度 数据

---

### 步骤4：打开前端

```bash
# 方式一：直接用浏览器打开
start frontend/index.html

# 方式二：使用简单静态服务器（推荐，避免CORS问题）
cd frontend && python -m http.server 8000
# 然后访问 http://127.0.0.1:8000
```

---

## 核心 API 接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/relics` | 获取文物列表 |
| GET | `/api/v1/relics/:id` | 获取文物详情 + 传感器 + 最新数据 |
| GET | `/api/v1/relics/:id/daily-stats?days=7` | 日粒度统计 |
| GET | `/api/v1/sensors/relic/:relic_id/latest` | 文物最新传感器数据 |
| GET | `/api/v1/sensors/:sensor_id/history?hours=24` | 单传感器历史 |
| **POST** | `/api/v1/sensors/upload` | **批量上传传感器数据** |
| GET | `/api/v1/alerts` | 告警列表 |
| GET | `/api/v1/alerts/stats` | 告警统计汇总 |
| **POST** | `/api/v1/algorithms/predict-scale-growth` | **结垢生长预测** |
| **POST** | `/api/v1/algorithms/predict-laser-cleaning` | **激光清洗参数优化** |
| GET | `/ws` | WebSocket 实时推送 |

### 上传传感器数据（EtherCAT 上报）

```json
POST /api/v1/sensors/upload
{
  "data": [
    {
      "sensor_id": 1,
      "relic_id": 1,
      "timestamp": "2025-06-12T10:00:00Z",
      "value": 1.85,
      "unit": "mm",
      "so2_concentration": 22.5,
      "humidity": 58.0,
      "temperature": 14.2
    }
  ]
}
```

---

## 核心算法详解

### 1. 结垢生长动力学模型

基于 **SO₂ 浓度 + 湿度 + 温度** 的经验动力学模型：

```
生长速率 = 基准速率 × f(SO₂) × f(Humidity) × f(Temperature)

其中：
  f(SO₂)      = (SO₂ × 10⁻³) ^ 0.7          (反应级数经验值)
  f(Humidity) = RH < 60%  ? 0.3 + 0.7(RH/60)³
                           : 1 + 2.5((RH-60)/40)²  (临界湿度效应)
  f(Temp)     = exp( Ea/R × (1/T₀ - 1/T) )    (Arrhenius方程, Ea=4000 J/mol)

叠加：日周期波动 (±10%)、生长饱和修正 (S型曲线趋近最大10mm)
```

代码位置：[scale_growth.go](file:///d:/SOLO-2/AI_solo_coder_task_A_055/backend/internal/algorithms/scale_growth.go)

---

### 2. 激光清洗阈值预测（热传导烧蚀模型）

基于**一维热传导方程**的解析解，搜索最优参数组合：

**烧蚀判据**：
```
能量密度 F > 烧蚀阈值 F_th (CaSO₄·2H₂O: ~1.2 J/cm²)
```

**烧蚀深度估算**：
```
δ = (F × η - F_th) × M / (ρ × L_v) × 10⁶

其中：
  F  = 实际能量密度 = 脉冲能量/光斑面积 × 重叠系数
  η  = 材料耦合效率 (硫酸钙 0.72, 方解石 0.85)
  M  = 摩尔质量 (136.14 g/mol)
  ρ  = 密度 (2.32 g/cm³)
  L_v = 汽化焓 (1.8 × 10⁶ J/kg)
```

**三维参数搜索空间**：
- 激光功率 P ∈ [50, 300] W
- 脉冲宽度 τ ∈ [200, 2000] ns  
- 扫描速度 v ∈ [10, 200] mm/s

**约束条件**：
- 光斑重叠率 ∈ [10%, 90%]
- 热穿透深度 ~ √(4ατ) ∈ [0.8d, 3d] (避免过度热影响)
- 阈值比 F/F_th ∈ [1.05, 3.0] (安全工作区间)

代码位置：[laser_cleaning.go](file:///d:/SOLO-2/AI_solo_coder_task_A_055/backend/internal/algorithms/laser_cleaning.go)

---

## 告警规则

| 指标 | 警告阈值 | 严重阈值 | 触发方式 |
|------|---------|---------|---------|
| **结垢厚度** | > 3.0 mm | > 4.5 mm | 钉钉 + WebSocket + 入库 |
| **表面粗糙度 Ra** | > 50.0 μm | > 75.0 μm | 钉钉 + WebSocket + 入库 |

**告警抑制**：同一传感器 1 小时内不重复告警。
**严重告警**：钉钉机器人自动 @所有人。

### 配置钉钉机器人

修改 `backend/config.yaml`：
```yaml
alert:
  dingtalk_webhook: "https://oapi.dingtalk.com/robot/send?access_token=你的TOKEN"
  dingtalk_secret: "你的加签SECRET"   # 可选，启用安全设置时填写
```

---

## 前端功能说明

### 1. 3D 模型视图（Three.js）
- **程序化生成** 佛像石像（基座→莲座→袈裟→躯干→头部→发髻→光背）
- 结垢厚度 **热力图覆盖**（传感器位置径向插值）
- 传感器实时标记（发光球体 + 脉冲动画）
- 支持线框模式 / OrbitControls 轨道控制

### 2. 结垢等高线图（Canvas）
- **Marching Squares 算法** 生成精确等高线
- 三种模式：仅等高线 / 填充+等高线 / 热力图
- 5~20 层级可调，实时标签显示数值
- 可导出 PNG 图片

### 3. 激光清洗模拟（Canvas + 粒子系统）
- 调用后端 API 计算最优参数（功率/脉宽/速度）
- **蛇形路径扫描**，实时显示光斑（激光光晕+核心+外围圈）
- **粒子特效**：烧蚀碎屑（橙黄火花 + 灰色粉尘）
- 进度条 + 预计剩余时间

### 4. 趋势分析
- 结垢厚度 + 粗糙度 双Y轴对比图
- SO₂浓度 / 相对湿度 / 环境温度 多参数关联分析
- 自动检测后端离线，使用内置算法生成模拟数据

### 5. 算法预测
- 基于输入的 SO₂/湿度/温度 进行生长预测
- 可视化 **告警触发时间点**（垂直线标注）
- 红色区域：超标危险区
- 增长速率/总量统计面板

---

## 关键数据量与性能估算

| 项目 | 数值 |
|------|------|
| 监测文物 | 10 处 |
| 超声测厚传感器 | 30 台 |
| 表面粗糙度仪 | 20 台 |
| 上报频率 | 每 2 小时 / 次 |
| 每年数据量 | 50 × 12 × 365 = 219,000 条 |
| ClickHouse 存储 | ~50MB/年（含TTL自动保留1年）|

---

## 常见问题

### Q1：前端无法连接后端？
- 检查后端 `http://127.0.0.1:8080/health` 是否返回 `{"status":"ok"}`
- 推荐使用 `python -m http.server` 启动前端，避免 file:// 协议 CORS 问题

### Q2：ClickHouse 连接失败？
- 确认容器端口 9000 已映射
- 修改 `backend/config.yaml` 中的 `clickhouse.host` 为实际地址
- 默认使用 TCP 原生协议（端口 9000），不是 HTTP 协议（8123）

### Q3：模拟器与后端启动顺序？
- 建议：ClickHouse → 后端 → 模拟器（模拟器启动会等待3秒自动回填历史数据）

### Q4：没有配置钉钉机器人？
- 不影响系统运行，钉钉推送会自动跳过
- WebSocket 实时告警仍可在前端看到

---

## 技术栈总结

| 层级 | 技术选型 | 版本 |
|------|---------|------|
| 数据库 | ClickHouse | 22+ |
| 后端语言 | Go | 1.21 |
| Web框架 | Gin | 1.9.1 |
| 数据库驱动 | clickhouse-go | 2.18.0 |
| 实时通信 | gorilla/websocket | 1.5.1 |
| 3D渲染 | Three.js | 0.160.0 (CDN) |
| 绘图 | HTML5 Canvas 2D | - |
| 配置 | Viper | 1.18.2 |
| 日志 | zap | 1.26.0 |
