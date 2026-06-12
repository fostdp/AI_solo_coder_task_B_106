const AlgorithmsUI = {
    chartCanvas: null,
    chartCtx: null,

    init() {
        this.chartCanvas = document.getElementById('prediction-chart');
        this.chartCtx = this.chartCanvas.getContext('2d');
        this.resizeCanvas();
        window.addEventListener('resize', () => this.resizeCanvas());

        document.getElementById('run-prediction').addEventListener('click', () => this.runPrediction());
    },

    resizeCanvas() {
        const parent = this.chartCanvas.parentElement;
        this.chartCanvas.width = parent.clientWidth - 40;
        this.chartCanvas.height = 350;
    },

    async runPrediction() {
        const hours = parseInt(document.getElementById('predict-hours').value);
        const initThickness = parseFloat(document.getElementById('init-thickness').value);
        const so2 = parseFloat(document.getElementById('predict-so2').value);
        const humidity = parseFloat(document.getElementById('predict-humidity').value);
        const temp = parseFloat(document.getElementById('predict-temp').value);

        const btn = document.getElementById('run-prediction');
        btn.disabled = true;
        btn.textContent = '预测计算中...';

        let result;
        try {
            result = await API.predictScaleGrowth({
                hours,
                initial_thickness: initThickness,
                so2_concentration: so2,
                humidity,
                temperature: temp
            });
        } catch (e) {
            result = this.localPredict(hours, initThickness, so2, humidity, temp);
        }

        this.drawChart(result, initThickness);
        btn.disabled = false;
        btn.textContent = '运行预测';
    },

    localPredict(hours, init, so2, humidity, temp) {
        const predicted = [];
        const so2Factor = Math.pow(so2 * 0.001, 0.7);
        const humidFactor = humidity > 60 
            ? 1 + 2.5 * Math.pow((humidity - 60) / 40, 2)
            : 0.3 + 0.7 * Math.pow(humidity / 60, 3);
        const tempFactor = Math.exp(4000 / 8.314 * (1/293.15 - 1/(temp + 273.15)));
        const rate = 0.00001 * so2Factor * humidFactor * tempFactor;

        for (let h = 0; h <= hours; h++) {
            const hourly = rate * (1 + 0.1 * Math.sin(2 * Math.PI * h / 24));
            const prev = h === 0 ? init : predicted[h - 1];
            const satFactor = 1 - Math.exp(-prev / 5);
            let v = prev + hourly * (1 - satFactor);
            v = Math.min(v, 10);
            predicted.push(v);
        }

        return {
            hours,
            initial_thickness: init,
            so2_concentration: so2,
            humidity,
            temperature: temp,
            predicted_thickness: predicted
        };
    },

    drawChart(result, initThickness) {
        const ctx = this.chartCtx;
        const w = this.chartCanvas.width;
        const h = this.chartCanvas.height;

        ctx.clearRect(0, 0, w, h);

        const padding = { left: 60, right: 30, top: 40, bottom: 55 };
        const chartW = w - padding.left - padding.right;
        const chartH = h - padding.top - padding.bottom;

        const data = result.predicted_thickness;
        if (!data || data.length === 0) return;

        const maxVal = Math.max(...data, 4);
        const maxT = Math.ceil(maxVal * 1.1 * 10) / 10;
        const maxH = result.hours;

        this.drawPredGrid(ctx, padding, chartW, chartH);

        const threshold = 3.0;
        const threshY = padding.top + chartH - (threshold / maxT) * chartH;
        ctx.save();
        ctx.strokeStyle = '#f44336';
        ctx.lineWidth = 1.5;
        ctx.setLineDash([8, 5]);
        ctx.beginPath();
        ctx.moveTo(padding.left, threshY);
        ctx.lineTo(padding.left + chartW, threshY);
        ctx.stroke();
        ctx.fillStyle = 'rgba(244, 67, 54, 0.1)';
        ctx.fillRect(padding.left, padding.top, chartW, threshY - padding.top);
        ctx.fillStyle = '#f44336';
        ctx.font = 'bold 11px sans-serif';
        ctx.textAlign = 'left';
        ctx.fillText(`⚠ 告警阈值: ${threshold}mm`, padding.left + 10, threshY - 8);
        ctx.restore();

        const criticalX = data.findIndex(v => v >= threshold);
        if (criticalX >= 0) {
            const cx = padding.left + (criticalX / maxH) * chartW;
            ctx.save();
            ctx.strokeStyle = 'rgba(255, 87, 34, 0.8)';
            ctx.lineWidth = 1.5;
            ctx.setLineDash([4, 4]);
            ctx.beginPath();
            ctx.moveTo(cx, padding.top);
            ctx.lineTo(cx, padding.top + chartH);
            ctx.stroke();

            const days = Math.floor(criticalX / 24);
            const hrs = criticalX % 24;
            ctx.fillStyle = '#ff5722';
            ctx.font = 'bold 11px sans-serif';
            ctx.textAlign = 'center';
            const label = days > 0 
                ? `预计${days}天${hrs}小时后超标`
                : `预计${hrs}小时后超标`;
            ctx.fillText(label, cx, padding.top + chartH + 38);
            ctx.restore();
        }

        const points = [];
        const step = Math.max(1, Math.floor(data.length / 500));
        for (let i = 0; i < data.length; i += step) {
            points.push({
                x: padding.left + (i / maxH) * chartW,
                y: padding.top + chartH - (data[i] / maxT) * chartH,
                v: data[i],
                t: i
            });
        }

        ctx.fillStyle = 'rgba(33, 150, 243, 0.2)';
        ctx.beginPath();
        ctx.moveTo(points[0].x, padding.top + chartH);
        points.forEach(p => ctx.lineTo(p.x, p.y));
        ctx.lineTo(points[points.length - 1].x, padding.top + chartH);
        ctx.closePath();
        ctx.fill();

        ctx.save();
        const grad = ctx.createLinearGradient(0, padding.top, 0, padding.top + chartH);
        grad.addColorStop(0, '#ff5722');
        grad.addColorStop(0.5, '#ffeb3b');
        grad.addColorStop(1, '#4caf50');
        ctx.strokeStyle = grad;
        ctx.lineWidth = 3;
        ctx.lineJoin = 'round';
        ctx.beginPath();
        ctx.moveTo(points[0].x, points[0].y);
        points.forEach(p => ctx.lineTo(p.x, p.y));
        ctx.stroke();
        ctx.restore();

        const endPoint = points[points.length - 1];
        ctx.fillStyle = '#1976d2';
        ctx.beginPath();
        ctx.arc(endPoint.x, endPoint.y, 6, 0, Math.PI * 2);
        ctx.fill();
        ctx.strokeStyle = 'white';
        ctx.lineWidth = 2;
        ctx.stroke();

        const startPoint = points[0];
        ctx.fillStyle = '#4caf50';
        ctx.beginPath();
        ctx.arc(startPoint.x, startPoint.y, 5, 0, Math.PI * 2);
        ctx.fill();

        this.drawPredAxis(ctx, padding, chartW, chartH, maxH, maxT);

        const growth = data[data.length - 1] - initThickness;
        const growthRate = growth / Math.max(maxH / 24, 0.1);
        this.drawPredLegend(ctx, w, {
            init: initThickness,
            final: data[data.length - 1],
            growth,
            growthRate,
            hours: maxH
        });
    },

    drawPredGrid(ctx, pad, w, h) {
        ctx.strokeStyle = 'rgba(255, 255, 255, 0.06)';
        ctx.lineWidth = 1;
        for (let i = 0; i <= 10; i++) {
            const y = pad.top + (h / 10) * i;
            ctx.beginPath();
            ctx.moveTo(pad.left, y);
            ctx.lineTo(pad.left + w, y);
            ctx.stroke();
        }
        for (let i = 0; i <= 10; i++) {
            const x = pad.left + (w / 10) * i;
            ctx.beginPath();
            ctx.moveTo(x, pad.top);
            ctx.lineTo(x, pad.top + h);
            ctx.stroke();
        }
    },

    drawPredAxis(ctx, pad, w, h, maxH, maxT) {
        ctx.fillStyle = 'rgba(255,255,255,0.6)';
        ctx.font = '10px monospace';
        ctx.textAlign = 'right';
        for (let i = 0; i <= 5; i++) {
            const val = (maxT / 5) * i;
            const y = pad.top + h - (h / 5) * i;
            ctx.fillText(val.toFixed(1) + ' mm', pad.left - 8, y + 3);
        }

        ctx.save();
        ctx.translate(15, pad.top + h / 2);
        ctx.rotate(-Math.PI / 2);
        ctx.textAlign = 'center';
        ctx.font = 'bold 12px sans-serif';
        ctx.fillStyle = '#e91e63';
        ctx.fillText('结垢厚度预测 (mm)', 0, 0);
        ctx.restore();

        ctx.fillStyle = 'rgba(255,255,255,0.6)';
        ctx.textAlign = 'center';
        const hDivs = 8;
        for (let i = 0; i <= hDivs; i++) {
            const hrs = (maxH / hDivs) * i;
            const x = pad.left + (w / hDivs) * i;
            const days = Math.floor(hrs / 24);
            const label = days > 0 ? `${days}d` : `${hrs}h`;
            ctx.fillText(label, x, pad.top + h + 18);
        }

        ctx.font = 'bold 12px sans-serif';
        ctx.fillStyle = '#2196f3';
        ctx.fillText('时间 (预测时长)', pad.left + w / 2, pad.top + h + 40);

        ctx.strokeStyle = 'rgba(255,255,255,0.2)';
        ctx.lineWidth = 1;
        ctx.beginPath();
        ctx.moveTo(pad.left, pad.top);
        ctx.lineTo(pad.left, pad.top + h);
        ctx.lineTo(pad.left + w, pad.top + h);
        ctx.stroke();
    },

    drawPredLegend(ctx, w, info) {
        const boxX = 20;
        const boxY = 5;
        const boxW = 360;
        const boxH = 28;

        ctx.fillStyle = 'rgba(10, 14, 26, 0.85)';
        ctx.fillRect(boxX, boxY, boxW, boxH);
        ctx.strokeStyle = 'rgba(102, 126, 234, 0.3)';
        ctx.lineWidth = 1;
        ctx.strokeRect(boxX, boxY, boxW, boxH);

        const items = [
            { label: '初始', value: info.init.toFixed(3) + 'mm', color: '#4caf50' },
            { label: '预测', value: info.final.toFixed(3) + 'mm', color: '#1976d2' },
            { label: '增长量', value: '+' + info.growth.toFixed(3) + 'mm', color: '#ff9800' },
            { label: '日均', value: info.growthRate.toFixed(4) + 'mm/d', color: '#e91e63' }
        ];

        let x = boxX + 12;
        ctx.font = '10px sans-serif';
        items.forEach(item => {
            ctx.fillStyle = item.color;
            ctx.fillRect(x, boxY + 8, 10, 10);
            ctx.fillStyle = 'rgba(255,255,255,0.5)';
            ctx.textAlign = 'left';
            ctx.fillText(item.label + ':', x + 14, boxY + 17);
            ctx.fillStyle = 'white';
            ctx.font = 'bold 11px monospace';
            ctx.fillText(item.value, x + 45, boxY + 17);
            ctx.font = '10px sans-serif';
            x += 85;
        });
    }
};
