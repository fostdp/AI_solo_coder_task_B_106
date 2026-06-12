const Trends = {
    currentRelicId: null,
    thicknessCanvas: null,
    thicknessCtx: null,
    envCanvas: null,
    envCtx: null,

    init() {
        this.thicknessCanvas = document.getElementById('trend-thickness');
        this.thicknessCtx = this.thicknessCanvas.getContext('2d');
        this.envCanvas = document.getElementById('trend-environment');
        this.envCtx = this.envCtx.getContext('2d');

        this.resizeCanvases();
        window.addEventListener('resize', () => this.resizeCanvases());
    },

    resizeCanvases() {
        [this.thicknessCanvas, this.envCanvas].forEach(c => {
            if (!c) return;
            const parent = c.parentElement;
            c.width = parent.clientWidth - 32;
            c.height = parent.clientHeight - 60;
        });
        if (this.currentRelicId) {
            this.render();
        }
    },

    async loadData(relicId) {
        this.currentRelicId = relicId;
        try {
            const stats = await API.getRelicDailyStats(relicId, 14);
            this.dailyStats = stats || [];
            this.render();
        } catch (e) {
            this.dailyStats = this.generateMockData(14);
            this.render();
        }
    },

    generateMockData(days) {
        const data = [];
        const now = new Date();
        let thickness = 0.5 + Math.random() * 0.5;
        let roughness = 8 + Math.random() * 5;
        for (let i = days - 1; i >= 0; i--) {
            const d = new Date(now);
            d.setDate(d.getDate() - i);
            thickness += 0.01 + Math.random() * 0.03;
            roughness += 0.1 + Math.random() * 0.3;
            if (Math.random() < 0.1) {
                thickness *= 0.9;
                roughness *= 0.92;
            }
            data.push({
                date: d,
                avg_thickness: thickness,
                max_thickness: thickness * (1.1 + Math.random() * 0.3),
                avg_roughness: roughness,
                max_roughness: roughness * (1.1 + Math.random() * 0.4),
                avg_so2: 15 + Math.sin(i / 3) * 8 + Math.random() * 5,
                avg_humidity: 45 + Math.sin(i / 2 + 1) * 15 + Math.random() * 5,
                avg_temperature: 12 + Math.sin(i / 3) * 10 + Math.random() * 2
            });
        }
        return data;
    },

    render() {
        this.renderThickness();
        this.renderEnvironment();
    },

    renderThickness() {
        const ctx = this.thicknessCtx;
        const w = this.thicknessCanvas.width;
        const h = this.thicknessCanvas.height;

        ctx.clearRect(0, 0, w, h);

        const data = this.dailyStats;
        if (!data || data.length === 0) {
            ctx.fillStyle = 'rgba(255,255,255,0.5)';
            ctx.font = '14px sans-serif';
            ctx.textAlign = 'center';
            ctx.fillText('暂无数据', w / 2, h / 2);
            return;
        }

        const padding = { left: 55, right: 20, top: 25, bottom: 40 };
        const chartW = w - padding.left - padding.right;
        const chartH = h - padding.top - padding.bottom;

        let maxT = 0, maxR = 0;
        data.forEach(d => {
            maxT = Math.max(maxT, d.max_thickness || 0);
            maxR = Math.max(maxR, d.max_roughness || 0);
        });
        maxT = Math.max(maxT * 1.2, 1);
        maxR = Math.max(maxR * 1.2, 10);

        this.drawGrid(ctx, padding, chartW, chartH, 5);
        this.drawYAxis(ctx, padding, chartH, maxT, '厚度 (mm)', '#e91e63');
        this.drawYAxisRight(ctx, padding, chartW, chartH, maxR, '粗糙度 (μm)', '#9c27b0');

        const xStep = chartW / Math.max(data.length - 1, 1);

        const avgTPts = data.map((d, i) => ({
            x: padding.left + i * xStep,
            y: padding.top + chartH - (d.avg_thickness / maxT) * chartH
        }));
        this.drawLine(ctx, avgTPts, '#e91e63', 2.5);
        this.drawArea(ctx, avgTPts, padding, chartH, 'rgba(233, 30, 99, 0.2)');

        const maxTPts = data.map((d, i) => ({
            x: padding.left + i * xStep,
            y: padding.top + chartH - (d.max_thickness / maxT) * chartH
        }));
        this.drawLine(ctx, maxTPts, 'rgba(255, 100, 150, 0.7)', 1.5, true);

        const avgRPts = data.map((d, i) => ({
            x: padding.left + i * xStep,
            y: padding.top + chartH - ((d.avg_roughness || 0) / maxR) * chartH
        }));
        this.drawLine(ctx, avgRPts, '#9c27b0', 2);

        this.drawXLabels(ctx, data, padding, chartW, chartH, xStep);

        this.drawLegend(ctx, w, [
            { color: '#e91e63', label: '平均厚度' },
            { color: 'rgba(255, 100, 150, 0.7)', label: '最大厚度', dashed: true },
            { color: '#9c27b0', label: '平均粗糙度' }
        ]);
    },

    renderEnvironment() {
        const ctx = this.envCtx;
        const w = this.envCanvas.width;
        const h = this.envCanvas.height;

        ctx.clearRect(0, 0, w, h);

        const data = this.dailyStats;
        if (!data || data.length === 0) return;

        const padding = { left: 55, right: 55, top: 25, bottom: 40 };
        const chartW = w - padding.left - padding.right;
        const chartH = h - padding.top - padding.bottom;

        let maxSO2 = 0, maxH = 0, minT = 100, maxT = -100;
        data.forEach(d => {
            maxSO2 = Math.max(maxSO2, d.avg_so2 || 0);
            maxH = Math.max(maxH, d.avg_humidity || 0);
            minT = Math.min(minT, d.avg_temperature || 0);
            maxT = Math.max(maxT, d.avg_temperature || 0);
        });
        maxSO2 = Math.max(maxSO2 * 1.2, 10);
        maxH = Math.max(maxH * 1.1, 50);
        const tRange = Math.max(maxT - minT, 10);
        minT -= tRange * 0.1;
        maxT += tRange * 0.1;

        this.drawGrid(ctx, padding, chartW, chartH, 5);

        const xStep = chartW / Math.max(data.length - 1, 1);

        const so2Pts = data.map((d, i) => ({
            x: padding.left + i * xStep,
            y: padding.top + chartH - ((d.avg_so2 || 0) / maxSO2) * chartH
        }));
        this.drawLine(ctx, so2Pts, '#ff9800', 2);
        this.drawBars(ctx, so2Pts, padding, chartH, 'rgba(255, 152, 0, 0.3)', xStep * 0.6);

        const humPts = data.map((d, i) => ({
            x: padding.left + i * xStep,
            y: padding.top + chartH - ((d.avg_humidity || 0) / maxH) * chartH
        }));
        this.drawLine(ctx, humPts, '#2196f3', 2);

        const tempPts = data.map((d, i) => {
            const tNorm = ((d.avg_temperature || 0) - minT) / (maxT - minT);
            return {
                x: padding.left + i * xStep,
                y: padding.top + chartH - tNorm * chartH
            };
        });
        this.drawLine(ctx, tempPts, '#f44336', 2.5);

        this.drawYAxis(ctx, padding, chartH, maxSO2, 'SO₂ (ppb)', '#ff9800');
        this.drawYAxisRight(ctx, padding, chartW, chartH, Math.round(maxT), '温度 (°C)', '#f44336');
        this.drawXLabels(ctx, data, padding, chartW, chartH, xStep);

        this.drawLegend(ctx, w, [
            { color: '#ff9800', label: 'SO₂ 浓度' },
            { color: '#2196f3', label: '相对湿度' },
            { color: '#f44336', label: '环境温度' }
        ]);
    },

    drawGrid(ctx, pad, w, h, divs) {
        ctx.strokeStyle = 'rgba(255, 255, 255, 0.05)';
        ctx.lineWidth = 1;
        for (let i = 0; i <= divs; i++) {
            const y = pad.top + (h / divs) * i;
            ctx.beginPath();
            ctx.moveTo(pad.left, y);
            ctx.lineTo(pad.left + w, y);
            ctx.stroke();
        }
        for (let i = 0; i <= divs; i++) {
            const x = pad.left + (w / divs) * i;
            ctx.beginPath();
            ctx.moveTo(x, pad.top);
            ctx.lineTo(x, pad.top + h);
            ctx.stroke();
        }
    },

    drawYAxis(ctx, pad, h, max, label, color) {
        ctx.strokeStyle = 'rgba(255,255,255,0.2)';
        ctx.lineWidth = 1;
        ctx.beginPath();
        ctx.moveTo(pad.left, pad.top);
        ctx.lineTo(pad.left, pad.top + h);
        ctx.stroke();

        ctx.fillStyle = color;
        ctx.font = '10px monospace';
        ctx.textAlign = 'right';
        for (let i = 0; i <= 5; i++) {
            const val = (max / 5) * i;
            const y = pad.top + h - (h / 5) * i;
            ctx.fillText(val.toFixed(1), pad.left - 6, y + 3);

            ctx.strokeStyle = 'rgba(255,255,255,0.1)';
            ctx.beginPath();
            ctx.moveTo(pad.left - 3, y);
            ctx.lineTo(pad.left, y);
            ctx.stroke();
        }

        ctx.save();
        ctx.translate(12, pad.top + h / 2);
        ctx.rotate(-Math.PI / 2);
        ctx.textAlign = 'center';
        ctx.font = '11px sans-serif';
        ctx.fillStyle = color;
        ctx.fillText(label, 0, 0);
        ctx.restore();
    },

    drawYAxisRight(ctx, pad, w, h, max, label, color) {
        const x = pad.left + w;
        ctx.strokeStyle = 'rgba(255,255,255,0.2)';
        ctx.beginPath();
        ctx.moveTo(x, pad.top);
        ctx.lineTo(x, pad.top + h);
        ctx.stroke();

        ctx.fillStyle = color;
        ctx.font = '10px monospace';
        ctx.textAlign = 'left';
        for (let i = 0; i <= 5; i++) {
            const val = (max / 5) * i;
            const y = pad.top + h - (h / 5) * i;
            ctx.fillText(val.toFixed(0), x + 6, y + 3);
        }

        ctx.save();
        ctx.translate(x + 42, pad.top + h / 2);
        ctx.rotate(Math.PI / 2);
        ctx.textAlign = 'center';
        ctx.font = '11px sans-serif';
        ctx.fillStyle = color;
        ctx.fillText(label, 0, 0);
        ctx.restore();
    },

    drawXLabels(ctx, data, pad, w, h, xStep) {
        ctx.fillStyle = 'rgba(255,255,255,0.5)';
        ctx.font = '10px sans-serif';
        ctx.textAlign = 'center';

        const labelCount = Math.min(data.length, 10);
        const labelStep = Math.ceil(data.length / labelCount);

        data.forEach((d, i) => {
            if (i % labelStep !== 0) return;
            const x = pad.left + i * xStep;
            const date = typeof d.date === 'string' ? new Date(d.date) : d.date;
            const label = `${date.getMonth() + 1}/${date.getDate()}`;
            ctx.fillText(label, x, pad.top + h + 18);
        });

        ctx.strokeStyle = 'rgba(255,255,255,0.2)';
        ctx.beginPath();
        ctx.moveTo(pad.left, pad.top + h);
        ctx.lineTo(pad.left + w, pad.top + h);
        ctx.stroke();
    },

    drawLine(ctx, points, color, width = 2, dashed = false) {
        if (points.length < 2) return;

        ctx.save();
        ctx.strokeStyle = color;
        ctx.lineWidth = width;
        ctx.lineJoin = 'round';
        ctx.lineCap = 'round';
        if (dashed) ctx.setLineDash([6, 4]);

        ctx.beginPath();
        ctx.moveTo(points[0].x, points[0].y);
        for (let i = 1; i < points.length; i++) {
            ctx.lineTo(points[i].x, points[i].y);
        }
        ctx.stroke();
        ctx.restore();

        points.forEach(p => {
            ctx.fillStyle = color;
            ctx.beginPath();
            ctx.arc(p.x, p.y, 2.5, 0, Math.PI * 2);
            ctx.fill();
        });
    },

    drawArea(ctx, points, pad, h, color) {
        if (points.length < 2) return;
        ctx.fillStyle = color;
        ctx.beginPath();
        ctx.moveTo(points[0].x, pad.top + h);
        points.forEach(p => ctx.lineTo(p.x, p.y));
        ctx.lineTo(points[points.length - 1].x, pad.top + h);
        ctx.closePath();
        ctx.fill();
    },

    drawBars(ctx, points, pad, h, color, width) {
        ctx.fillStyle = color;
        points.forEach(p => {
            const barH = pad.top + h - p.y;
            ctx.fillRect(p.x - width / 2, p.y, width, barH);
        });
    },

    drawLegend(ctx, w, items) {
        ctx.font = '11px sans-serif';
        const itemW = 100;
        const startX = w - items.length * itemW - 10;
        let x = startX;

        items.forEach(item => {
            ctx.fillStyle = item.color;
            if (item.dashed) {
                ctx.save();
                ctx.setLineDash([4, 2]);
                ctx.strokeStyle = item.color;
                ctx.lineWidth = 2;
                ctx.beginPath();
                ctx.moveTo(x, 12);
                ctx.lineTo(x + 20, 12);
                ctx.stroke();
                ctx.restore();
            } else {
                ctx.fillRect(x, 8, 20, 3);
            }
            ctx.fillStyle = 'rgba(255,255,255,0.7)';
            ctx.textAlign = 'left';
            ctx.fillText(item.label, x + 25, 13);
            x += itemW;
        });
    }
};
