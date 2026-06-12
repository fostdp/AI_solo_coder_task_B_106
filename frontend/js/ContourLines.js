(function(window) {
    'use strict';

    const ContourLines = {
        EDGE_TABLE: [
            0, 9, 12, 5, 6, 15, 10, 3,
            10, 3, 15, 6, 5, 12, 9, 0
        ],

        VERTEX_OFFSETS: [
            [0, 0], [1, 0], [1, 1], [0, 1]
        ],

        generateDataGrid(gridSize, latestData) {
            const dataGrid = [];
            const w = gridSize;
            const h = gridSize;

            const sensorPositions = latestData.filter(d => d.latest_unit === 'mm').map((d, i) => ({
                x: 0.15 + (i * 0.25) % 0.7,
                y: 0.2 + Math.floor(i * 0.25 / 0.7) * 0.3,
                value: d.latest_value,
                radius: 0.25 + (0.15 * ((d.sensor_id * 7 + 3) % 10) / 10)
            }));

            if (sensorPositions.length === 0) {
                sensorPositions.push({ x: 0.5, y: 0.5, value: 0.8, radius: 0.3 });
            }

            for (let y = 0; y < h; y++) {
                const row = [];
                for (let x = 0; x < w; x++) {
                    const nx = x / (w - 1);
                    const ny = y / (h - 1);

                    let sumWeights = 0;
                    let sumValue = 0;

                    sensorPositions.forEach(sp => {
                        const dx = nx - sp.x;
                        const dy = ny - sp.y;
                        const dist = Math.sqrt(dx * dx + dy * dy);
                        const weight = Math.exp(-(dist * dist) / (sp.radius * sp.radius * 0.5));
                        sumValue += sp.value * weight;
                        sumWeights += weight;
                    });

                    const base = sumWeights > 0 ? sumValue / sumWeights : 0.5;
                    const noise = (Math.sin(nx * 15) + Math.cos(ny * 12) + Math.sin((nx + ny) * 8)) * 0.05;
                    const edgeFade = 1 - Math.pow(Math.max(Math.abs(nx - 0.5), Math.abs(ny - 0.5)) * 2, 3) * 0.3;

                    row.push(Math.max(0, (base + noise) * edgeFade));
                }
                dataGrid.push(row);
            }

            return dataGrid;
        },

        getValue(dataGrid, x, y) {
            const ix = Math.floor(x);
            const iy = Math.floor(y);
            const fx = x - ix;
            const fy = y - iy;

            const h = dataGrid.length;
            const w = dataGrid[0].length;

            const x0 = Math.max(0, Math.min(w - 2, ix));
            const y0 = Math.max(0, Math.min(h - 2, iy));

            const v00 = dataGrid[y0][x0];
            const v10 = dataGrid[y0][x0 + 1];
            const v01 = dataGrid[y0 + 1][x0];
            const v11 = dataGrid[y0 + 1][x0 + 1];

            return this.lerp(this.lerp(v00, v10, fx), this.lerp(v01, v11, fx), fy);
        },

        lerp(a, b, t) {
            return a + (b - a) * t;
        },

        marchingSquares(dataGrid, isoLevel, max, scaleX, scaleY) {
            const lines = [];
            const h = dataGrid.length;
            const w = dataGrid[0].length;

            for (let y = 0; y < h - 1; y++) {
                for (let x = 0; x < w - 1; x++) {
                    const cellLines = this.cellContour(dataGrid, x, y, isoLevel, max, scaleX, scaleY);
                    if (cellLines.length > 0) {
                        lines.push(...cellLines);
                    }
                }
            }

            return lines;
        },

        cellContour(dataGrid, x, y, isoLevel, max, scaleX, scaleY) {
            const v = [
                dataGrid[y][x],
                dataGrid[y][x + 1],
                dataGrid[y + 1][x + 1],
                dataGrid[y + 1][x]
            ];

            let idx = 0;
            for (let i = 0; i < 4; i++) {
                if (v[i] > isoLevel) {
                    idx |= (1 << i);
                }
            }

            const edges = this.EDGE_TABLE[idx];
            if (edges === 0 || edges === 15) {
                return [];
            }

            const pts = [];
            for (let e = 0; e < 4; e++) {
                if (edges & (1 << e)) {
                    pts.push(this.edgePoint(dataGrid, x, y, e, isoLevel, scaleX, scaleY));
                }
            }

            const lines = [];
            for (let i = 0; i < pts.length; i += 2) {
                if (i + 1 < pts.length) {
                    lines.push([pts[i], pts[i + 1]]);
                }
            }

            return lines;
        },

        edgePoint(dataGrid, x, y, edge, isoLevel, scaleX, scaleY) {
            const v = [
                dataGrid[y][x],
                dataGrid[y][x + 1],
                dataGrid[y + 1][x + 1],
                dataGrid[y + 1][x]
            ];

            let t;
            let px, py;

            switch (edge) {
                case 0:
                    t = (isoLevel - v[0]) / (v[1] - v[0] || 1e-12);
                    px = x + t;
                    py = y;
                    break;
                case 1:
                    t = (isoLevel - v[1]) / (v[2] - v[1] || 1e-12);
                    px = x + 1;
                    py = y + t;
                    break;
                case 2:
                    t = (isoLevel - v[3]) / (v[2] - v[3] || 1e-12);
                    px = x + t;
                    py = y + 1;
                    break;
                case 3:
                    t = (isoLevel - v[0]) / (v[3] - v[0] || 1e-12);
                    px = x;
                    py = y + t;
                    break;
                default:
                    px = x;
                    py = y;
            }

            return [px * scaleX, py * scaleY];
        },

        createHeatmapImage(data, width, height, gridSize, max) {
            const imgData = document.createElement('canvas').getContext('2d').createImageData(width, height);

            for (let py = 0; py < height; py++) {
                const gy = (py / height) * (gridSize - 1);
                for (let px = 0; px < width; px++) {
                    const gx = (px / width) * (gridSize - 1);
                    const val = this.getValue(data, gx, gy);
                    const ratio = Math.min(val / max, 1);

                    let r, g, b;
                    if (ratio < 0.2) {
                        r = 0;
                        g = 230;
                        b = 118;
                    } else if (ratio < 0.4) {
                        const t = (ratio - 0.2) / 0.2;
                        r = Math.floor(t * 118);
                        g = 230;
                        b = Math.floor(118 * (1 - t));
                    } else if (ratio < 0.6) {
                        const t = (ratio - 0.4) / 0.2;
                        r = 118 + Math.floor(t * 137);
                        g = 230 - Math.floor(t * 30);
                        b = 0;
                    } else if (ratio < 0.8) {
                        const t = (ratio - 0.6) / 0.2;
                        r = 255;
                        g = 200 - Math.floor(t * 150);
                        b = 0;
                    } else {
                        const t = (ratio - 0.8) / 0.2;
                        r = 255;
                        g = 50 - Math.floor(t * 30);
                        b = Math.floor(t * 50);
                    }

                    const idx = (py * width + px) * 4;
                    imgData.data[idx] = r;
                    imgData.data[idx + 1] = g;
                    imgData.data[idx + 2] = b;
                    imgData.data[idx + 3] = 220;
                }
            }

            return imgData;
        }
    };

    window.ContourLines = ContourLines;
})(window);
