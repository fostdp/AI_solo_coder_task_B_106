const API_BASE = 'http://127.0.0.1:8080/api/v1';

const API = {
    async getRelics() {
        const res = await fetch(`${API_BASE}/relics`);
        return res.json();
    },

    async getRelicDetail(id) {
        const res = await fetch(`${API_BASE}/relics/${id}`);
        return res.json();
    },

    async getRelicDailyStats(id, days = 7) {
        const res = await fetch(`${API_BASE}/relics/${id}/daily-stats?days=${days}`);
        return res.json();
    },

    async getLatestSensorData(relicId) {
        const res = await fetch(`${API_BASE}/sensors/relic/${relicId}/latest`);
        return res.json();
    },

    async getSensorHistory(sensorId, hours = 24) {
        const res = await fetch(`${API_BASE}/sensors/${sensorId}/history?hours=${hours}`);
        return res.json();
    },

    async uploadSensorData(batch) {
        const res = await fetch(`${API_BASE}/sensors/upload`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ data: batch })
        });
        return res.json();
    },

    async getAlerts(days = 7, limit = 100) {
        const res = await fetch(`${API_BASE}/alerts?days=${days}&limit=${limit}`);
        return res.json();
    },

    async getAlertsByRelic(relicId, limit = 50) {
        const res = await fetch(`${API_BASE}/alerts/relic/${relicId}?limit=${limit}`);
        return res.json();
    },

    async getAlertStats(days = 30) {
        const res = await fetch(`${API_BASE}/alerts/stats?days=${days}`);
        return res.json();
    },

    async predictScaleGrowth(data) {
        const res = await fetch(`${API_BASE}/algorithms/predict-scale-growth`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data)
        });
        return res.json();
    },

    async predictLaserCleaning(data) {
        const res = await fetch(`${API_BASE}/algorithms/predict-laser-cleaning`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data)
        });
        return res.json();
    },

    async createCleaningRecord(record) {
        const res = await fetch(`${API_BASE}/cleaning/records`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(record)
        });
        return res.json();
    },

    async getCleaningRecords(relicId = null, limit = 100) {
        const url = relicId 
            ? `${API_BASE}/cleaning/records?relic_id=${relicId}&limit=${limit}`
            : `${API_BASE}/cleaning/records?limit=${limit}`;
        const res = await fetch(url);
        return res.json();
    },

    async getCleaningOptLog(limit = 50) {
        const res = await fetch(`${API_BASE}/cleaning/opt-log?limit=${limit}`);
        return res.json();
    },

    async planTSPPath(data) {
        const res = await fetch(`${API_BASE}/advanced/plan-tsp-path`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data)
        });
        return res.json();
    },

    async predictRoughness(data) {
        const res = await fetch(`${API_BASE}/advanced/predict-roughness`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data)
        });
        return res.json();
    },

    async predictRescaling(data) {
        const res = await fetch(`${API_BASE}/advanced/predict-rescaling`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data)
        });
        return res.json();
    },

    async simulateRobot(data) {
        const res = await fetch(`${API_BASE}/advanced/simulate-robot`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data)
        });
        return res.json();
    }
};
