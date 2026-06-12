const Contour = {
    canvas: null,
    ctx: null,
    gridSize: 80,
    dataGrid: [],
    relicData: null,
    _dirty: true,
    _cachedFilled: null,
    _cachedFilledMax: 0,
    _cachedLines: null,
    _cachedLinesLevels: 0,
    _cachedLinesMax: 0,
    _cachedCanvasSize: '',

    init() {
        this.canvas = document.getElementById('contour-canvas');
        this.ctx = this.canvas.getContext('2d');
        this.resize();
        window.addEventListener('resize', () => this.resize());
    },

    resize() {
        if (!this.canvas) return;
        const container = this.canvas.parentElement;
        const w = container.clientWidth;
        const h = container.clientHeight;
        this.canvas.width = w;
        this.canvas.height = h;
        this._dirty = true;
        this._cachedFilled = null;
        this._cachedLines = null;
    },

    generateDataGrid(latestData) {
        this.dataGrid = ContourLines.generateDataGrid(this.gridSize, latestData);
        this._dirty = true;
        this._cachedFilled = null;
        this._cachedLines = null;
    },

    getValue(x, y) {
        return ContourLines.getValue(this.dataGrid, x, y);
    },

    renderFilled(max, cellW, cellH) {
        const w = this.canvas.width;
        const h = this.canvas.height;
        const sizeKey = `${w}x${h}`;

        if (!this._dirty && this._cachedFilled && this._cachedFilledMax === max && this._cachedCanvasSize === sizeKey) {
            this.ctx.putImageData(this._cachedFilled, 0, 0);
            return;
        }

        const imgData = ContourLines.createHeatmapImage(this.dataGrid, w, h, this.gridSize, max);

        this._cachedFilled = imgData;
        this._cachedFilledMax = max;
        this._cachedCanvasSize = sizeKey;
        this.ctx.putImageData(imgData, 0, 0);
    },

    renderHeatmap(max, cellW, cellH) {
        const w = this.canvas.width;
        const h = this.canvas.height;

        const imgData = this.ctx.createImageData(w, h);
        const data = imgData.data;

        for (let py = 0; py < h; py++) {
            const gy = (py / h) * (this.gridSize - 1);
            for (let px = 0; px < w; px++) {
                const gx = (px / w) * (this.gridSize - 1);
                const val = ContourLines.getValue(this.dataGrid, gx, gy);
                const ratio = Math.min(val / max, 1);

                const hue = 120 - ratio * 120;
                const saturation = 85;
                const lightness = 20 + ratio * 30;
                const rgb = this.hslToRgb(hue / 360, saturation / 100, lightness / 100);

                const idx = (py * w + px) * 4;
                data[idx] = rgb[0];
                data[idx + 1] = rgb[1];
                data[idx + 2] = rgb[2];
                data[idx + 3] = 255;
            }
        }

        this.ctx.putImageData(imgData, 0, 0);
    },

    hslToRgb(h, s, l) {
        let r, g, b;
        if (s === 0) {
            r = g = b = l;
        } else {
            const hue2rgb = (p, q, t) => {
                if (t < 0) t += 1;
                if (t > 1) t -= 1;
                if (t < 1/6) return p + (q - p) * 6 * t;
                if (t < 1/2) return q;
                if (t < 2/3) return p + (q - p) * (2/3 - t) * 6;
                return p;
            };
            const q = l < 0.5 ? l * (1 + s) : l + s - l * s;
            const p = 2 * l - q;
            r = hue2rgb(p, q, h + 1/3);
            g = hue2rgb(p, q, h);
            b = hue2rgb(p, q, h - 1/3);
        }
        return [Math.round(r * 255), Math.round(g * 255), Math.round(b * 255)];
    },

    renderContourLines(levels, max, scaleX, scaleY) {
        const maxVal = max;

        if (!this._dirty && this._cachedLines && this._cachedLinesLevels === levels && this._cachedLinesMax === max) {
            this._drawCachedLines(this._cachedLines, maxVal);
            return;
        }

        const allLines = [];
        for (let i = 0; i < levels; i++) {
            const level = (i + 1) * (maxVal / (levels + 1));
            const ratio = level / maxVal;
            const lines = ContourLines.marchingSquares(this.dataGrid, level, max, scaleX, scaleY);
            allLines.push({ level, ratio, lines });
        }

        this._cachedLines = allLines;
        this._cachedLinesLevels = levels;
        this._cachedLinesMax = max;
        this._drawCachedLines(allLines, maxVal);
    },

    _drawCachedLines(allLines, maxVal) {
        allLines.forEach(({ level, ratio, lines }) => {
            this.ctx.strokeStyle = `hsla(${120 - ratio * 120}, 90%, 60%, ${0.4 + ratio * 0.5})`;
            this.ctx.lineWidth = ratio > 0.7 ? 2 : 1;
            this.ctx.beginPath();
            lines.forEach(line => {
                this.ctx.moveTo(line[0][0], line[0][1]);
                this.ctx.lineTo(line[1][0], line[1][1]);
            });
            this.ctx.stroke();
        });

        allLines.forEach(({ level, ratio, lines }, i) => {
            if (i % 2 === 1) {
                this.ctx.fillStyle = `hsla(${120 - ratio * 120}, 90%, 80%, 0.9)`;
                this.ctx.font = '9px monospace';
                for (let j = 0; j < lines.length; j += 8) {
                    const line = lines[j];
                    if (line) {
                        const mx = (line[0][0] + line[1][0]) / 2;
                        const my = (line[0][1] + line[1][1]) / 2;
                        this.ctx.fillText(`${level.toFixed(2)}mm`, mx, my);
                    }
                }
            }
        });
    },

    renderFrame(w, h) {
        this.ctx.strokeStyle = '#3a4460';
        this.ctx.lineWidth = 2;
        this.ctx.strokeRect(0, 0, w, h);
    },

    renderScaleBar(w, h, max) {
        const barX = w - 40;
        const barY = 20;
        const barW = 20;
        const barH = h - 40;

        const gradient = this.ctx.createLinearGradient(barX, barY + barH, barX, barY);
        gradient.addColorStop(0, '#00e676');
        gradient.addColorStop(0.25, '#76ff03');
        gradient.addColorStop(0.5, '#ffd600');
        gradient.addColorStop(0.75, '#ff6f00');
        gradient.addColorStop(1, '#d50000');

        this.ctx.fillStyle = gradient;
        this.ctx.fillRect(barX, barY, barW, barH);

        this.ctx.fillStyle = '#e0e0e0';
        this.ctx.font = '10px monospace';
        this.ctx.textAlign = 'left';
        const steps = 5;
        for (let i = 0; i <= steps; i++) {
            const y = barY + barH - (i / steps) * barH;
            const val = (i / steps) * max;
            this.ctx.fillText(val.toFixed(1), barX + barW + 5, y + 3);
            this.ctx.strokeStyle = 'rgba(255,255,255,0.3)';
            this.ctx.beginPath();
            this.ctx.moveTo(barX, y);
            this.ctx.lineTo(barX + barW, y);
            this.ctx.stroke();
        }

        this.ctx.fillStyle = '#ffffff';
        this.ctx.font = 'bold 10px monospace';
        this.ctx.fillText('mm', barX + barW + 5, barY - 5);
    },

    renderSensorPositions(latestData, w, h) {
        const positions = [
            { x: 0.2, y: 0.25 }, { x: 0.45, y: 0.3 },
            { x: 0.7, y: 0.25 }, { x: 0.25, y: 0.5 },
            { x: 0.5, y: 0.55 }, { x: 0.75, y: 0.5 },
            { x: 0.3, y: 0.75 }, { x: 0.55, y: 0.8 },
        ];

        positions.forEach((pos, i) => {
            const x = pos.x * w;
            const y = pos.y * h;

            this.ctx.beginPath();
            this.ctx.arc(x, y, 8, 0, Math.PI * 2);
            this.ctx.fillStyle = 'rgba(244, 67, 54, 0.3)';
            this.ctx.fill();
            this.ctx.strokeStyle = '#f44336';
            this.ctx.lineWidth = 1.5;
            this.ctx.stroke();

            this.ctx.fillStyle = '#ffffff';
            this.ctx.font = '8px monospace';
            this.ctx.textAlign = 'center';
            this.ctx.fillText(`S${i + 1}`, x, y + 3);
        });
    },

    render(relicData) {
        if (relicData) {
            this.relicData = relicData;
            this.generateDataGrid(relicData.latest_data || []);
        }
        if (!this.relicData) return;

        const latestData = this.relicData.latest_data || [];

        const levels = parseInt(document.getElementById('contour-levels').value);
        const mode = document.getElementById('contour-mode').value;

        this.ctx.clearRect(0, 0, this.canvas.width, this.canvas.height);

        const w = this.canvas.width;
        const h = this.canvas.height;
        const cellW = w / this.gridSize;
        const cellH = h / this.gridSize;

        let max = 0;
        this.dataGrid.forEach(row => row.forEach(v => max = Math.max(max, v)));
        max = Math.max(max, 1);

        if (mode === 'filled') {
            this.renderFilled(max, cellW, cellH);
        } else if (mode === 'heatmap') {
            this.renderHeatmap(max, cellW, cellH);
        }

        if (mode !== 'heatmap' || levels > 0) {
            this.renderContourLines(levels, max, cellW, cellH);
        }

        this.renderFrame(w, h);
        this.renderScaleBar(w, h, max);
        this.renderSensorPositions(latestData, w, h);

        this._dirty = false;
    },

    exportPNG() {
        const link = document.createElement('a');
        link.download = `contour_${Date.now()}.png`;
        link.href = this.canvas.toDataURL('image/png');
        link.click();
    }
};
