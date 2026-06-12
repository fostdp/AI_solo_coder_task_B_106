const AdvancedFeatures = {
    robot: null,
    currentPath: null,
    currentRelicId: 1,

    init(viewer) {
        this.viewer = viewer;
        this._bindEvents();
        console.log('[AdvancedFeatures] 初始化完成');
    },

    _bindEvents() {
        const btnPlan = document.getElementById('btn-plan-path');
        if (btnPlan) btnPlan.addEventListener('click', () => this.planPath());

        const btnSimRobot = document.getElementById('btn-sim-robot');
        if (btnSimRobot) btnSimRobot.addEventListener('click', () => this.startRobotSimulation());

        const btnStopRobot = document.getElementById('btn-stop-robot');
        if (btnStopRobot) btnStopRobot.addEventListener('click', () => this.stopRobot());

        const btnPredictRoughness = document.getElementById('btn-predict-roughness');
        if (btnPredictRoughness) btnPredictRoughness.addEventListener('click', () => this.predictRoughness());

        const btnPredictRescaling = document.getElementById('btn-predict-rescaling');
        if (btnPredictRescaling) btnPredictRescaling.addEventListener('click', () => this.predictRescaling());
    },

    _generateSamplePoints(relicId) {
        const n = 12 + (relicId % 5) * 3;
        const points = [];
        for (let i = 0; i < n; i++) {
            const angle = (i / n) * Math.PI * 2 + Math.random() * 0.4;
            const radius = 3 + Math.random() * 3;
            const x = Math.cos(angle) * radius + (Math.random() - 0.5) * 1.5;
            const z = Math.sin(angle) * radius + (Math.random() - 0.5) * 1.5;
            points.push({
                id: i,
                x: x,
                y: 0,
                z: z,
                thickness: 0.5 + Math.random() * 3.5,
                area: 0.8 + Math.random() * 2.5,
                priority: Math.random() > 0.7 ? 2 : 1
            });
        }
        return points;
    },

    async planPath() {
        const relicId = this.currentRelicId;
        const algorithm = document.getElementById('path-algorithm').value;
        const robotSpeed = parseFloat(document.getElementById('robot-speed').value) || 50;

        const points = this._generateSamplePoints(relicId);

        const resultDiv = document.getElementById('tsp-result');
        resultDiv.innerHTML = '<div class="loading">正在规划路径...</div>';

        try {
            const result = await API.planTSPPath({
                relic_id: relicId,
                points: points,
                robot_speed: robotSpeed,
                algorithm: algorithm,
                start_point: { x: -8, y: 0, z: 0, thickness: 0, area: 0, priority: 0, id: -1 }
            });

            this.currentPath = result.ordered_points;

            let html = `
                <div class="result-card">
                    <h4>✓ 路径规划完成</h4>
                    <div class="result-grid">
                        <div><span>算法：</span><strong>${result.algorithm || 'two_opt'}</strong></div>
                        <div><span>清洗点：</span><strong>${result.ordered_points.length} 个</strong></div>
                        <div><span>总距离：</span><strong>${result.total_distance.toFixed(2)} mm</strong></div>
                        <div><span>预计时间：</span><strong>${(result.total_time_seconds / 60).toFixed(1)} 分钟</strong></div>
                        <div><span>迭代次数：</span><strong>${result.iterations}</strong></div>
                    </div>
                </div>
            `;
            resultDiv.innerHTML = html;

            this._renderPathChart(result.ordered_points);
            this._showPathOnModel(result.ordered_points);

            document.getElementById('btn-sim-robot').disabled = false;
        } catch (err) {
            resultDiv.innerHTML = `<div class="error">路径规划失败: ${err.message}</div>`;
            console.error(err);
        }
    },

    _renderPathChart(points) {
        const canvas = document.getElementById('path-canvas');
        if (!canvas) return;
        const ctx = canvas.getContext('2d');
        const W = canvas.width, H = canvas.height;

        ctx.clearRect(0, 0, W, H);
        ctx.fillStyle = '#1a1d24';
        ctx.fillRect(0, 0, W, H);

        const padding = 30;
        const xs = points.map(p => p.x);
        const zs = points.map(p => p.z);
        const minX = Math.min(...xs) - 1, maxX = Math.max(...xs) + 1;
        const minZ = Math.min(...zs) - 1, maxZ = Math.max(...zs) + 1;
        const scaleX = (W - padding * 2) / (maxX - minX);
        const scaleZ = (H - padding * 2) / (maxZ - minZ);
        const scale = Math.min(scaleX, scaleZ);

        const tx = (x) => padding + (x - minX) * scale;
        const tz = (z) => padding + (z - minZ) * scale;

        ctx.strokeStyle = '#2a3040';
        ctx.lineWidth = 1;
        for (let i = 0; i <= 10; i++) {
            ctx.beginPath();
            ctx.moveTo(padding, padding + i * (H - padding * 2) / 10);
            ctx.lineTo(W - padding, padding + i * (H - padding * 2) / 10);
            ctx.stroke();
            ctx.beginPath();
            ctx.moveTo(padding + i * (W - padding * 2) / 10, padding);
            ctx.lineTo(padding + i * (W - padding * 2) / 10, H - padding);
            ctx.stroke();
        }

        if (points.length > 1) {
            ctx.beginPath();
            ctx.strokeStyle = '#00ffff';
            ctx.lineWidth = 2.5;
            ctx.setLineDash([8, 4]);
            for (let i = 0; i < points.length; i++) {
                const px = tx(points[i].x), pz = tz(points[i].z);
                if (i === 0) ctx.moveTo(px, pz);
                else ctx.lineTo(px, pz);
            }
            ctx.stroke();
            ctx.setLineDash([]);

            for (let i = 0; i < points.length - 1; i++) {
                const mx = (tx(points[i].x) + tx(points[i + 1].x)) / 2;
                const mz = (tz(points[i].z) + tz(points[i + 1].z)) / 2;
                const angle = Math.atan2(tz(points[i + 1].z) - tz(points[i].z), tx(points[i + 1].x) - tx(points[i].x));
                ctx.fillStyle = '#00ffff';
                ctx.save();
                ctx.translate(mx, mz);
                ctx.rotate(angle);
                ctx.beginPath();
                ctx.moveTo(0, 0);
                ctx.lineTo(-6, -4);
                ctx.lineTo(-6, 4);
                ctx.closePath();
                ctx.fill();
                ctx.restore();
            }
        }

        points.forEach((p, i) => {
            const hue = i / Math.max(points.length - 1, 1);
            const color = `hsl(${hue * 300}, 100%, 55%)`;
            const px = tx(p.x), pz = tz(p.z);
            ctx.beginPath();
            ctx.fillStyle = color;
            ctx.arc(px, pz, 7, 0, Math.PI * 2);
            ctx.fill();
            ctx.beginPath();
            ctx.fillStyle = '#fff';
            ctx.arc(px, pz, 4, 0, Math.PI * 2);
            ctx.fill();
            ctx.fillStyle = '#000';
            ctx.font = 'bold 10px sans-serif';
            ctx.textAlign = 'center';
            ctx.textBaseline = 'middle';
            ctx.fillText(i + 1, px, pz);
        });

        ctx.fillStyle = '#8899aa';
        ctx.font = '11px sans-serif';
        ctx.textAlign = 'left';
        ctx.fillText('X (mm)', padding, H - 8);
        ctx.save();
        ctx.translate(10, H / 2);
        ctx.rotate(-Math.PI / 2);
        ctx.fillText('Z (mm)', 0, 0);
        ctx.restore();
    },

    _showPathOnModel(points) {
        if (!this.viewer || !this.viewer.scene) return;

        if (!this.robot) {
            this.robot = new CleaningRobot(this.viewer.scene);
        }
        this.robot.setPosition(-8, 0, 0);
        this.robot.showPath(points);
    },

    async startRobotSimulation() {
        if (!this.currentPath || this.currentPath.length === 0) {
            alert('请先进行路径规划');
            return;
        }

        if (!this.robot) {
            this.robot = new CleaningRobot(this.viewer.scene);
        }

        const progressDiv = document.getElementById('robot-progress');
        const speedFactor = parseFloat(document.getElementById('speed-factor').value) || 2;

        progressDiv.innerHTML = `<div class="loading">🤖 机器人开始清洗作业...</div>`;

        this.robot.followPath(
            this.currentPath,
            () => {
                progressDiv.innerHTML = `
                    <div class="result-card success">
                        <h4>✓ 清洗作业完成</h4>
                        <p>所有 ${this.currentPath.length} 个清洗点处理完毕</p>
                        <p>清洗范围已在3D模型上用绿色标记显示</p>
                    </div>
                `;
            },
            (progress, currentIdx) => {
                const pct = Math.round(progress * 100);
                progressDiv.innerHTML = `
                    <div class="result-card working">
                        <h4>🤖 正在执行清洗...</h4>
                        <div class="progress-bar">
                            <div class="progress-fill" style="width:${pct}%"></div>
                        </div>
                        <p>进度：${pct}% (${currentIdx + 1}/${this.currentPath.length})</p>
                        <p>当前清洗点：#${currentIdx >= 0 && this.currentPath[currentIdx] ? this.currentPath[currentIdx].id : '--'}，激光 ${this.robot && this.robot.laserActive ? '<span style="color:#ff4444">开启中</span>' : '<span style="color:#666">待机</span>'}</p>
                    </div>
                `;
            }
        );
    },

    stopRobot() {
        if (this.robot) {
            this.robot.stop();
        }
        const progressDiv = document.getElementById('robot-progress');
        if (progressDiv) {
            progressDiv.innerHTML = '<div class="result-card warning"><p>机器人已停止</p></div>';
        }
    },

    async predictRoughness() {
        const relicId = this.currentRelicId;
        const energyDensity = parseFloat(document.getElementById('roughness-energy').value) || 1.8;
        const laserPower = parseFloat(document.getElementById('roughness-power').value) || 200;
        const pulseDuration = parseFloat(document.getElementById('roughness-pulse').value) || 1000;
        const scanSpeed = parseFloat(document.getElementById('roughness-speed').value) || 80;
        const initialRoughness = parseFloat(document.getElementById('roughness-initial').value) || 30;
        const overlapRate = parseFloat(document.getElementById('roughness-overlap').value) || 0.5;

        const cs = parseFloat(document.getElementById('mineral-cs').value) || 0.55;
        const cc = parseFloat(document.getElementById('mineral-cc').value) || 0.25;
        const dol = parseFloat(document.getElementById('mineral-dol').value) || 0.12;
        const sil = parseFloat(document.getElementById('mineral-sil').value) || 0.08;
        const total = cs + cc + dol + sil;

        const resultDiv = document.getElementById('roughness-result');
        resultDiv.innerHTML = '<div class="loading">正在预测表面形貌...</div>';

        try {
            const result = await API.predictRoughness({
                relic_id: relicId,
                energy_density: energyDensity,
                laser_power: laserPower,
                pulse_duration: pulseDuration,
                scan_speed: scanSpeed,
                initial_roughness: initialRoughness,
                overlap_rate: overlapRate,
                mineral_composition: {
                    calcium_sulfate: cs / total,
                    calcite: cc / total,
                    dolomite: dol / total,
                    silicate: sil / total,
                    gypsum: 0
                }
            });

            const riskColor = result.risk_level === 'high' ? '#ff4444' : result.risk_level === 'medium' ? '#ffaa00' : '#44dd44';
            const riskLabel = result.risk_level === 'high' ? '⚠️ 高风险' : result.risk_level === 'medium' ? '⚡ 中风险' : '✓ 低风险';

            let html = `
                <div class="result-card">
                    <h4>预测结果</h4>
                    <div class="big-number" style="color:${riskColor}">
                        ${result.predicted_roughness.toFixed(2)} <small>μm (Ra)</small>
                    </div>
                    <div>风险等级：<strong style="color:${riskColor}">${riskLabel}</strong></div>
                    <div>预测范围：${result.roughness_range[0].toFixed(2)} ~ ${result.roughness_range[1].toFixed(2)} μm</div>
                    <div>模型置信度：${(result.confidence * 100).toFixed(0)}%</div>
                </div>
                <div class="result-card">
                    <h4>特征重要性</h4>
            `;

            const features = Object.entries(result.feature_importance).sort((a, b) => b[1] - a[1]);
            const featureNames = {
                'energy_density': '能量密度',
                'laser_power': '激光功率',
                'pulse_duration': '脉冲宽度',
                'scan_speed': '扫描速度',
                'initial_roughness': '初始粗糙度',
                'overlap_rate': '光斑重叠率',
                'mineral_composition': '矿物成分'
            };

            features.forEach(([k, v]) => {
                const pct = Math.round(v * 100);
                const name = featureNames[k] || k;
                html += `
                    <div class="feature-row">
                        <span>${name}</span>
                        <div class="feature-bar-wrap"><div class="feature-bar" style="width:${pct}%"></div></div>
                        <span class="feature-val">${pct}%</span>
                    </div>
                `;
            });

            html += '</div>';

            this._drawRoughnessChart(result);

            resultDiv.innerHTML = html;
        } catch (err) {
            resultDiv.innerHTML = `<div class="error">预测失败: ${err.message}</div>`;
            console.error(err);
        }
    },

    _drawRoughnessChart(result) {
        const canvas = document.getElementById('roughness-canvas');
        if (!canvas) return;
        const ctx = canvas.getContext('2d');
        const W = canvas.width, H = canvas.height;
        ctx.clearRect(0, 0, W, H);
        ctx.fillStyle = '#1a1d24';
        ctx.fillRect(0, 0, W, H);

        const pred = result.predicted_roughness;
        const low = result.roughness_range[0];
        const high = result.roughness_range[1];
        const maxR = Math.max(high * 1.2, 60);

        const pad = 40;
        const cy = H - pad;
        const baseY = pad + 20;
        const scaleY = (cy - baseY) / maxR;

        ctx.strokeStyle = '#3a4050';
        ctx.lineWidth = 1;
        for (let v = 0; v <= maxR; v += 10) {
            const y = cy - v * scaleY;
            ctx.beginPath();
            ctx.moveTo(pad, y);
            ctx.lineTo(W - pad, y);
            ctx.stroke();
            ctx.fillStyle = '#889';
            ctx.font = '10px sans-serif';
            ctx.textAlign = 'right';
            ctx.fillText(v + ' μm', pad - 5, y + 3);
        }

        const cx = W / 2;

        ctx.fillStyle = 'rgba(100,100,255,0.25)';
        ctx.fillRect(cx - 80, cy - high * scaleY, 160, (high - low) * scaleY);
        ctx.strokeStyle = 'rgba(100,100,255,0.5)';
        ctx.strokeRect(cx - 80, cy - high * scaleY, 160, (high - low) * scaleY);

        ctx.fillStyle = '#ff6633';
        ctx.beginPath();
        ctx.arc(cx, cy - pred * scaleY, 10, 0, Math.PI * 2);
        ctx.fill();
        ctx.fillStyle = '#fff';
        ctx.font = 'bold 11px sans-serif';
        ctx.textAlign = 'center';
        ctx.fillText(pred.toFixed(1) + ' μm', cx, cy - pred * scaleY - 18);

        ctx.strokeStyle = '#ff4444';
        ctx.setLineDash([5, 5]);
        ctx.beginPath();
        ctx.moveTo(pad, cy - 50 * scaleY);
        ctx.lineTo(W - pad, cy - 50 * scaleY);
        ctx.stroke();
        ctx.setLineDash([]);
        ctx.fillStyle = '#ff4444';
        ctx.textAlign = 'left';
        ctx.fillText('⚠️ 阈值 50 μm', W - pad - 95, cy - 50 * scaleY - 5);

        ctx.fillStyle = '#889';
        ctx.font = '10px sans-serif';
        ctx.textAlign = 'center';
        ctx.fillText('清洗前', cx - 140, cy - 5);
        ctx.fillText('清洗后', cx, cy - 5);
    },

    async predictRescaling() {
        const relicId = this.currentRelicId;
        const hours = parseInt(document.getElementById('rescale-hours').value) || 24;
        const postCleaning = document.getElementById('rescale-post-clean').checked;

        const resultDiv = document.getElementById('rescale-result');
        resultDiv.innerHTML = '<div class="loading">ARIMA模型正在预测二次结垢...</div>';

        try {
            const history = [];
            let base = 0.03;
            for (let i = 0; i < 20; i++) {
                base += 0.004 + Math.random() * 0.003;
                history.push(parseFloat(base.toFixed(4)));
            }

            const result = await API.predictRescaling({
                relic_id: relicId,
                history_data: history,
                hours: hours,
                so2_concentration: 25,
                humidity: 65,
                temperature: 16,
                post_cleaning: postCleaning
            });

            const riskColor = result.risk_level === 'high' ? '#ff4444' : result.risk_level === 'medium' ? '#ffaa00' : '#44dd44';
            const riskLabel = result.risk_level === 'high' ? '⚠️ 高风险' : result.risk_level === 'medium' ? '⚡ 中风险' : '✓ 低风险';

            let html = `
                <div class="result-card">
                    <h4>二次结垢预测结果</h4>
                    <div>风险等级：<strong style="color:${riskColor}">${riskLabel}</strong></div>
                    <div>ARIMA 参数：(p=${result.arima_params[0]}, d=${result.arima_params[1]}, q=${result.arima_params[2]})</div>
                    <div>模型置信度：${(result.confidence * 100).toFixed(0)}%</div>
                    <div>警告阈值：${result.warning_threshold.toFixed(3)} mm</div>
            `;
            if (result.warning_trigger_hour) {
                html += `<div style="color:#ff4444">⚠️ 预计 <strong>${result.warning_trigger_hour} 小时</strong> 后达到警告阈值</div>`;
            } else {
                html += `<div style="color:#44dd44">✓ ${hours}小时内不会达到警告阈值</div>`;
            }
            html += '</div>';

            this._drawRescalingChart(result, history);

            resultDiv.innerHTML = html;
        } catch (err) {
            resultDiv.innerHTML = `<div class="error">预测失败: ${err.message}</div>`;
            console.error(err);
        }
    },

    _drawRescalingChart(result, history) {
        const canvas = document.getElementById('rescale-canvas');
        if (!canvas) return;
        const ctx = canvas.getContext('2d');
        const W = canvas.width, H = canvas.height;
        ctx.clearRect(0, 0, W, H);
        ctx.fillStyle = '#1a1d24';
        ctx.fillRect(0, 0, W, H);

        const pad = 45;
        const nHist = history.length;
        const nPred = result.hours.length;
        const totalN = nHist + nPred;
        const maxV = Math.max(...result.predicted_thickness, ...history, result.warning_threshold) * 1.15;
        const minV = 0;

        const scaleX = (W - pad * 2) / (totalN - 1);
        const scaleY = (H - pad * 2) / (maxV - minV);

        const tx = (i) => pad + i * scaleX;
        const ty = (v) => H - pad - (v - minV) * scaleY;

        ctx.strokeStyle = '#2a3040';
        ctx.lineWidth = 1;
        for (let v = 0; v <= maxV; v += maxV / 5) {
            ctx.beginPath();
            ctx.moveTo(pad, ty(v));
            ctx.lineTo(W - pad, ty(v));
            ctx.stroke();
            ctx.fillStyle = '#889';
            ctx.font = '10px sans-serif';
            ctx.textAlign = 'right';
            ctx.fillText(v.toFixed(3) + ' mm', pad - 5, ty(v) + 3);
        }

        ctx.strokeStyle = '#ff4444';
        ctx.setLineDash([6, 4]);
        ctx.lineWidth = 1.5;
        ctx.beginPath();
        ctx.moveTo(pad, ty(result.warning_threshold));
        ctx.lineTo(W - pad, ty(result.warning_threshold));
        ctx.stroke();
        ctx.setLineDash([]);
        ctx.fillStyle = '#ff4444';
        ctx.textAlign = 'left';
        ctx.fillText('⚠️ 警告阈值 ' + result.warning_threshold.toFixed(3) + 'mm', pad + 5, ty(result.warning_threshold) - 5);

        if (nHist > 1) {
            ctx.strokeStyle = '#66aaff';
            ctx.lineWidth = 2;
            ctx.beginPath();
            for (let i = 0; i < nHist; i++) {
                const x = tx(i), y = ty(history[i]);
                if (i === 0) ctx.moveTo(x, y);
                else ctx.lineTo(x, y);
            }
            ctx.stroke();

            ctx.fillStyle = '#66aaff';
            for (let i = 0; i < nHist; i++) {
                ctx.beginPath();
                ctx.arc(tx(i), ty(history[i]), 3, 0, Math.PI * 2);
                ctx.fill();
            }
        }

        if (nPred > 1) {
            const grad = ctx.createLinearGradient(tx(nHist), 0, tx(totalN - 1), 0);
            grad.addColorStop(0, '#ffaa33');
            grad.addColorStop(1, '#ff3333');
            ctx.strokeStyle = grad;
            ctx.lineWidth = 2.5;
            ctx.setLineDash([4, 3]);
            ctx.beginPath();
            for (let i = 0; i < nPred; i++) {
                const x = tx(nHist + i), y = ty(result.predicted_thickness[i]);
                if (i === 0) ctx.moveTo(x, y);
                else ctx.lineTo(x, y);
            }
            ctx.stroke();
            ctx.setLineDash([]);

            ctx.fillStyle = '#ff6633';
            for (let i = 0; i < nPred; i++) {
                ctx.beginPath();
                ctx.arc(tx(nHist + i), ty(result.predicted_thickness[i]), 3.5, 0, Math.PI * 2);
                ctx.fill();
            }
        }

        ctx.strokeStyle = '#00ff88';
        ctx.lineWidth = 1;
        ctx.setLineDash([2, 3]);
        ctx.beginPath();
        ctx.moveTo(tx(nHist), pad);
        ctx.lineTo(tx(nHist), H - pad);
        ctx.stroke();
        ctx.setLineDash([]);
        ctx.fillStyle = '#00ff88';
        ctx.textAlign = 'center';
        ctx.fillText('现在 →', tx(nHist), H - pad + 18);

        ctx.fillStyle = '#889';
        ctx.font = '10px sans-serif';
        ctx.textAlign = 'center';
        ctx.fillText('时间步 (过去→未来)', W / 2, H - 8);
    }
};
