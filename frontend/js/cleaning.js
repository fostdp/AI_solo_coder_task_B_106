const CleaningSim = {
    canvas: null,
    ctx: null,
    surfaceData: [],
    particles: [],
    running: false,
    rafId: null,
    laserX: 0,
    laserY: 0,
    laserPath: [],
    startTime: 0,
    cleanedCount: 0,
    totalArea: 0,

    init() {
        this.canvas = document.getElementById('cleaning-canvas');
        this.ctx = this.canvas.getContext('2d');
        this.resizeCanvas();
        window.addEventListener('resize', () => this.resizeCanvas());

        document.getElementById('sim-power').addEventListener('input', (e) => {
            document.getElementById('sim-power-value').textContent = e.target.value + ' W';
        });
        document.getElementById('sim-speed').addEventListener('input', (e) => {
            document.getElementById('sim-speed-value').textContent = e.target.value + ' mm/s';
        });

        document.getElementById('calc-params').addEventListener('click', () => this.calcOptimalParams());
        document.getElementById('start-sim').addEventListener('click', () => this.start());
        document.getElementById('stop-sim').addEventListener('click', () => this.stop());

        this.generateSurface();
        this.draw();
    },

    resizeCanvas() {
        const container = this.canvas.parentElement;
        this.canvas.width = container.clientWidth;
        this.canvas.height = container.clientHeight;
        if (this.surfaceData.length > 0) {
            this.draw();
        }
    },

    generateSurface() {
        const w = 200;
        const h = 200;
        this.surfaceData = [];
        this.totalArea = w * h;
        this.cleanedCount = 0;

        const hotspots = [];
        const numHotspots = 5 + Math.floor(Math.random() * 4);
        for (let i = 0; i < numHotspots; i++) {
            hotspots.push({
                x: Math.random(),
                y: Math.random(),
                radius: 0.1 + Math.random() * 0.2,
                strength: 1.5 + Math.random() * 2.5
            });
        }

        for (let y = 0; y < h; y++) {
            const row = [];
            for (let x = 0; x < w; x++) {
                const nx = x / (w - 1);
                const ny = y / (h - 1);

                let val = 0.5 + Math.random() * 0.3;

                hotspots.forEach(hs => {
                    const dx = nx - hs.x;
                    const dy = ny - hs.y;
                    const dist = Math.sqrt(dx * dx + dy * dy);
                    val += hs.strength * Math.exp(-(dist * dist) / (hs.radius * hs.radius * 0.3));
                });

                for (let f = 1; f < 6; f++) {
                    val += Math.sin(nx * f * 8 + ny * f * 5) * 
                            Math.cos(ny * f * 6 - nx * f * 3) * 0.08 / f;
                }

                row.push({
                    original: val,
                    remaining: Math.max(0.1, val),
                    cleaned: false
                });
            }
            this.surfaceData.push(row);
        }

        this.laserPath = this.generatePath();
    },

    generatePath() {
        const path = [];
        const stepsY = 40;
        const h = 200;

        for (let row = 0; row < stepsY; row++) {
            const y = (row / (stepsY - 1)) * (h - 1);
            const startX = row % 2 === 0 ? 0 : h - 1;
            const endX = row % 2 === 0 ? h - 1 : 0;
            const step = row % 2 === 0 ? 1 : -1;

            for (let x = startX; row % 2 === 0 ? x <= endX : x >= endX; x += step) {
                path.push({ x, y });
            }
        }
        return path;
    },

    async calcOptimalParams() {
        const targetThickness = parseFloat(document.getElementById('target-thickness').value);
        const materialType = document.getElementById('material-type').value;

        const btn = document.getElementById('calc-params');
        btn.disabled = true;
        btn.textContent = '计算中...';

        try {
            const result = await API.predictLaserCleaning({
                target_thickness: targetThickness,
                material_type: materialType
            });

            const container = document.getElementById('cleaning-result');
            container.innerHTML = `
                <div style="margin-bottom:10px;color:var(--secondary);font-weight:600;">
                    ✓ 最优参数计算完成
                </div>
                <div class="result-row">
                    <span>激光功率:</span>
                    <span style="color:var(--accent-gold);font-weight:bold">${result.optimal_power.toFixed(1)} W</span>
                </div>
                <div class="result-row">
                    <span>脉冲宽度:</span>
                    <span style="color:var(--accent-gold);font-weight:bold">${result.optimal_pulse.toFixed(0)} ns</span>
                </div>
                <div class="result-row">
                    <span>扫描速度:</span>
                    <span style="color:var(--accent-gold);font-weight:bold">${result.optimal_speed.toFixed(1)} mm/s</span>
                </div>
                <div class="result-row">
                    <span>能量密度:</span>
                    <span>${result.predicted_energy_density.toFixed(2)} J/cm²</span>
                </div>
                <div class="result-row">
                    <span>烧蚀阈值:</span>
                    <span>${result.ablation_threshold.toFixed(2)} J/cm²</span>
                </div>
                <div class="result-row">
                    <span>预测烧蚀深度:</span>
                    <span style="color:var(--primary);font-weight:bold">${result.predicted_depth.toFixed(3)} mm</span>
                </div>
                <div class="result-row">
                    <span>置信度:</span>
                    <span>${(result.confidence * 100).toFixed(1)}%</span>
                </div>
                <div class="result-row warning" style="margin-top:8px;">
                    <span>⚠️ ${result.safety_warning}</span>
                </div>
            `;

            document.getElementById('sim-power').value = Math.round(result.optimal_power);
            document.getElementById('sim-power-value').textContent = Math.round(result.optimal_power) + ' W';
            document.getElementById('sim-speed').value = Math.round(result.optimal_speed);
            document.getElementById('sim-speed-value').textContent = Math.round(result.optimal_speed) + ' mm/s';

        } catch (e) {
            this.useFallbackParams(targetThickness);
        } finally {
            btn.disabled = false;
            btn.textContent = '计算最优参数';
        }
    },

    useFallbackParams(targetThickness) {
        const power = 150 + targetThickness * 50;
        const speed = 80 - targetThickness * 10;
        const container = document.getElementById('cleaning-result');
        container.innerHTML = `
            <div style="margin-bottom:10px;color:var(--warning);font-weight:600;">
                ⚡ 使用本地估算参数（离线模式）
            </div>
            <div class="result-row">
                <span>推荐功率:</span>
                <span style="color:var(--accent-gold);font-weight:bold">${power.toFixed(0)} W</span>
            </div>
            <div class="result-row">
                <span>推荐速度:</span>
                <span style="color:var(--accent-gold);font-weight:bold">${Math.max(10, speed).toFixed(0)} mm/s</span>
            </div>
            <div class="result-row">
                <span>目标厚度:</span>
                <span>${targetThickness.toFixed(2)} mm</span>
            </div>
        `;
        document.getElementById('sim-power').value = Math.round(power);
        document.getElementById('sim-power-value').textContent = Math.round(power) + ' W';
        document.getElementById('sim-speed').value = Math.max(10, Math.round(speed));
        document.getElementById('sim-speed-value').textContent = Math.max(10, Math.round(speed)) + ' mm/s';
    },

    start() {
        if (this.running) return;
        this.running = true;
        this.startTime = Date.now();
        this.particles = [];
        this.currentPathIdx = 0;

        const power = parseFloat(document.getElementById('sim-power').value);
        const speed = parseFloat(document.getElementById('sim-speed').value);
        this.stepInterval = Math.max(1, Math.round(200 / speed));
        this.cleaningRadius = 5 + power / 100;
        this.cleaningStrength = 0.05 + (power / 500) * 0.15;

        Stats.showToast('清洗模拟启动', `功率 ${power}W / 速度 ${speed}mm/s`, 'info');
        this.animate();
    },

    stop() {
        this.running = false;
        if (this.rafId) cancelAnimationFrame(this.rafId);
        document.getElementById('sim-progress').textContent = Math.floor(this.cleanedCount / this.totalArea * 100);
    },

    animate() {
        if (!this.running) return;

        const speed = parseFloat(document.getElementById('sim-speed').value);
        const stepsPerFrame = Math.max(1, Math.floor(speed / 50));

        for (let s = 0; s < stepsPerFrame; s++) {
            if (this.currentPathIdx >= this.laserPath.length) {
                this.running = false;
                Stats.showToast('清洗完成', `已清洗 ${Math.floor(this.cleanedCount / this.totalArea * 100)}% 区域`, 'info');
                break;
            }

            const pos = this.laserPath[this.currentPathIdx];
            this.laserX = pos.x;
            this.laserY = pos.y;
            this.cleanAt(pos.x, pos.y);
            this.currentPathIdx++;
        }

        this.updateParticles();
        this.draw();
        this.updateProgress();

        if (this.running) {
            this.rafId = requestAnimationFrame(() => this.animate());
        }
    },

    cleanAt(cx, cy) {
        const r = this.cleaningRadius;
        const r2 = r * r;

        for (let dy = -r; dy <= r; dy++) {
            const y = Math.floor(cy + dy);
            if (y < 0 || y >= 200) continue;
            for (let dx = -r; dx <= r; dx++) {
                const x = Math.floor(cx + dx);
                if (x < 0 || x >= 200) continue;
                const dist2 = dx * dx + dy * dy;
                if (dist2 > r2) continue;

                const falloff = 1 - Math.sqrt(dist2) / r;
                const strength = this.cleaningStrength * falloff * falloff;
                const cell = this.surfaceData[y][x];

                if (cell.remaining > 0.05) {
                    cell.remaining = Math.max(0.01, cell.remaining - strength);

                    if (falloff > 0.6 && Math.random() < strength * 20) {
                        this.spawnParticle(cx, cy);
                    }
                }

                if (!cell.cleaned && cell.remaining <= cell.original * 0.1) {
                    cell.cleaned = true;
                    this.cleanedCount++;
                }
            }
        }
    },

    spawnParticle(x, y) {
        const canvasScaleX = this.canvas.width / 200;
        const canvasScaleY = this.canvas.height / 200;

        for (let i = 0; i < 3; i++) {
            const angle = Math.random() * Math.PI * 2;
            const speed = 0.5 + Math.random() * 2;
            this.particles.push({
                x: x * canvasScaleX,
                y: y * canvasScaleY,
                vx: Math.cos(angle) * speed,
                vy: Math.sin(angle) * speed - 2 - Math.random() * 2,
                life: 1,
                decay: 0.015 + Math.random() * 0.02,
                size: 1 + Math.random() * 3,
                color: Math.random() > 0.5 
                    ? { r: 255, g: 200 + Math.random() * 55, b: 100 }
                    : { r: 180, g: 180, b: 180 }
            });
        }
    },

    updateParticles() {
        this.particles = this.particles.filter(p => {
            p.x += p.vx;
            p.y += p.vy;
            p.vy += 0.05;
            p.vx *= 0.98;
            p.life -= p.decay;
            return p.life > 0;
        });
    },

    updateProgress() {
        const progress = Math.floor(this.cleanedCount / this.totalArea * 100);
        document.getElementById('sim-progress').textContent = progress;

        const elapsed = (Date.now() - this.startTime) / 1000;
        if (progress > 0 && progress < 100) {
            const remaining = (100 - progress) * (elapsed / progress);
            const min = Math.floor(remaining / 60);
            const sec = Math.floor(remaining % 60);
            document.getElementById('sim-eta').textContent = 
                `${min.toString().padStart(2, '0')}:${sec.toString().padStart(2, '0')}`;
        } else if (progress >= 100) {
            document.getElementById('sim-eta').textContent = '完成 ✓';
        }
    },

    draw() {
        const ctx = this.ctx;
        const w = this.canvas.width;
        const h = this.canvas.height;

        ctx.clearRect(0, 0, w, h);

        const cellW = w / 200;
        const cellH = h / 200;

        const imgData = ctx.createImageData(w, h);
        for (let py = 0; py < h; py++) {
            const gy = Math.floor((py / h) * 200);
            for (let px = 0; px < w; px++) {
                const gx = Math.floor((px / w) * 200);
                const cell = this.surfaceData[gy]?.[gx];
                if (!cell) continue;

                const ratio = cell.remaining / Math.max(0.1, cell.original);

                let r, g, b;
                if (ratio > 0.9) {
                    r = 140 + Math.random() * 20;
                    g = 120 + Math.random() * 20;
                    b = 100 + Math.random() * 20;
                } else if (ratio > 0.5) {
                    const t = (ratio - 0.5) / 0.4;
                    r = 100 + t * 40;
                    g = 110 + t * 10;
                    b = 120 + t * (-20);
                    r += (Math.random() - 0.5) * 15;
                } else {
                    const t = Math.min(1, ratio / 0.5);
                    r = 70 + t * 30;
                    g = 75 + t * 35;
                    b = 80 + t * 40;
                }

                const idx = (py * w + px) * 4;
                imgData.data[idx] = Math.max(0, Math.min(255, r));
                imgData.data[idx + 1] = Math.max(0, Math.min(255, g));
                imgData.data[idx + 2] = Math.max(0, Math.min(255, b));
                imgData.data[idx + 3] = 255;
            }
        }
        ctx.putImageData(imgData, 0, 0);

        if (this.running || this.cleanedCount > 0) {
            const canvasScaleX = w / 200;
            const canvasScaleY = h / 200;

            const lx = this.laserX * canvasScaleX;
            const ly = this.laserY * canvasScaleY;
            const laserR = this.cleaningRadius * canvasScaleX;

            const glowGrad = ctx.createRadialGradient(lx, ly, 0, lx, ly, laserR * 3);
            glowGrad.addColorStop(0, 'rgba(255, 100, 50, 0.6)');
            glowGrad.addColorStop(0.3, 'rgba(255, 150, 50, 0.3)');
            glowGrad.addColorStop(1, 'rgba(255, 200, 50, 0)');
            ctx.fillStyle = glowGrad;
            ctx.beginPath();
            ctx.arc(lx, ly, laserR * 3, 0, Math.PI * 2);
            ctx.fill();

            ctx.fillStyle = 'rgba(255, 255, 200, 0.9)';
            ctx.beginPath();
            ctx.arc(lx, ly, laserR * 0.3, 0, Math.PI * 2);
            ctx.fill();

            ctx.strokeStyle = 'rgba(255, 50, 0, 0.9)';
            ctx.lineWidth = 2;
            ctx.beginPath();
            ctx.arc(lx, ly, laserR, 0, Math.PI * 2);
            ctx.stroke();
        }

        this.particles.forEach(p => {
            ctx.globalAlpha = p.life;
            ctx.fillStyle = `rgb(${p.color.r}, ${p.color.g}, ${p.color.b})`;
            ctx.beginPath();
            ctx.arc(p.x, p.y, p.size * p.life, 0, Math.PI * 2);
            ctx.fill();
        });
        ctx.globalAlpha = 1;

        this.drawOverlay(w, h);
    },

    drawOverlay(w, h) {
        const ctx = this.ctx;

        ctx.strokeStyle = 'rgba(102, 126, 234, 0.4)';
        ctx.lineWidth = 2;
        ctx.strokeRect(5, 5, w - 10, h - 10);

        if (this.cleanedCount > 0) {
            const progress = this.cleanedCount / this.totalArea;
            const barH = 8;
            const barW = w - 40;
            const barX = 20;
            const barY = h - 20;

            ctx.fillStyle = 'rgba(0, 0, 0, 0.5)';
            ctx.fillRect(barX, barY, barW, barH);

            const grad = ctx.createLinearGradient(barX, 0, barX + barW, 0);
            grad.addColorStop(0, '#4caf50');
            grad.addColorStop(progress, '#8bc34a');
            ctx.fillStyle = grad;
            ctx.fillRect(barX, barY, barW * progress, barH);

            ctx.strokeStyle = 'rgba(255, 255, 255, 0.3)';
            ctx.lineWidth = 1;
            ctx.strokeRect(barX, barY, barW, barH);
        }

        const legend = [
            { color: '#8c7a64', label: '严重结垢' },
            { color: '#6e7982', label: '轻度结垢' },
            { color: '#4a5460', label: '已清洗' }
        ];

        ctx.font = '11px sans-serif';
        let lx = 15;
        let ly = 15;

        legend.forEach(l => {
            ctx.fillStyle = l.color;
            ctx.fillRect(lx, ly, 14, 14);
            ctx.strokeStyle = 'rgba(255,255,255,0.3)';
            ctx.strokeRect(lx, ly, 14, 14);
            ctx.fillStyle = 'rgba(255,255,255,0.8)';
            ctx.fillText(l.label, lx + 20, ly + 11);
            ly += 20;
        });
    },

    reset() {
        this.stop();
        this.generateSurface();
        this.particles = [];
        this.currentPathIdx = 0;
        document.getElementById('sim-progress').textContent = '0';
        document.getElementById('sim-eta').textContent = '--:--';
        this.draw();
    }
};
