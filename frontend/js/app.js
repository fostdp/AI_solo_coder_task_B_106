const App = {
    relics: [],
    currentRelicId: null,
    currentRelicDetail: null,

    async init() {
        this.updateTime();
        setInterval(() => this.updateTime(), 1000);

        this.setupTabs();

        WS.connect();
        WS.on('alert', (data) => this.handleWSAlert(data));
        WS.on('sensor_data', (data) => this.handleWSSensorData(data));
        WS.on('stats', (data) => this.handleWSStats(data));

        ThreeViewer.init();
        Contour.init();
        CleaningSim.init();
        Trends.init();
        AlgorithmsUI.init();
        AdvancedFeatures.init(ThreeViewer);

        await this.loadInitialData();

        setInterval(() => this.refreshData(), 30000);
        setInterval(() => this.refreshAlerts(), 60000);
    },

    updateTime() {
        const now = new Date();
        const str = now.toLocaleString('zh-CN', {
            year: 'numeric',
            month: '2-digit',
            day: '2-digit',
            hour: '2-digit',
            minute: '2-digit',
            second: '2-digit',
            hour12: false
        });
        document.getElementById('current-time').textContent = str;
    },

    setupTabs() {
        document.querySelectorAll('.tab').forEach(tab => {
            tab.addEventListener('click', () => {
                document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
                document.querySelectorAll('.tab-content').forEach(c => c.classList.remove('active'));
                tab.classList.add('active');
                const tabId = 'tab-' + tab.dataset.tab;
                document.getElementById(tabId).classList.add('active');

                if (tab.dataset.tab === 'contour' && this.currentRelicDetail) {
                    setTimeout(() => Contour.render(this.currentRelicDetail), 50);
                }
                if (tab.dataset.tab === 'trends' && this.currentRelicId) {
                    setTimeout(() => Trends.resizeCanvases(), 50);
                }
                if (tab.dataset.tab === 'model') {
                    setTimeout(() => ThreeViewer.onResize(), 50);
                }
                if (tab.dataset.tab === 'cleaning') {
                    setTimeout(() => CleaningSim.resizeCanvas(), 50);
                }
                if (tab.dataset.tab === 'algorithm') {
                    setTimeout(() => AlgorithmsUI.resizeCanvas(), 50);
                }
            });
        });
    },

    async loadInitialData() {
        try {
            this.relics = await API.getRelics();
            if (!this.relics || this.relics.length === 0) {
                this.relics = this.getMockRelics();
            }
        } catch (e) {
            console.warn('Using mock relic data:', e.message);
            this.relics = this.getMockRelics();
        }

        this.renderRelicList();

        try {
            const stats = await API.getAlertStats();
            if (stats) Stats.renderAlertStats(stats);
        } catch (e) {}

        try {
            const alerts = await API.getAlerts(7, 20);
            if (alerts) Stats.renderAlerts(alerts);
        } catch (e) {}

        try {
            const records = await API.getCleaningRecords(null, 10);
            if (records) Stats.renderCleaningLogs(records);
        } catch (e) {}

        if (this.relics.length > 0) {
            this.selectRelic(this.relics[0].id);
        }
    },

    getMockRelics() {
        return [
            { id: 1, name: '云冈石窟-第20窟大佛', location: '山西大同' },
            { id: 2, name: '乐山大佛', location: '四川乐山' },
            { id: 3, name: '龙门石窟-卢舍那大佛', location: '河南洛阳' },
            { id: 4, name: '敦煌莫高窟-第96窟', location: '甘肃敦煌' },
            { id: 5, name: '麦积山石窟-第44窟', location: '甘肃天水' },
            { id: 6, name: '大足石刻-宝顶山', location: '重庆大足' },
            { id: 7, name: '响堂山石窟-北响堂', location: '河北邯郸' },
            { id: 8, name: '天龙山石窟', location: '山西太原' },
            { id: 9, name: '巩义石窟寺', location: '河南巩义' },
            { id: 10, name: '须弥山石窟', location: '宁夏固原' }
        ];
    },

    renderRelicList() {
        const container = document.getElementById('relic-list');
        container.innerHTML = this.relics.map(r => `
            <div class="relic-item" data-id="${r.id}">
                <div class="relic-name">${r.name}</div>
                <div class="relic-location">📍 ${r.location}</div>
            </div>
        `).join('');

        container.querySelectorAll('.relic-item').forEach(item => {
            item.addEventListener('click', () => {
                const id = parseInt(item.dataset.id);
                this.selectRelic(id);
            });
        });
    },

    async selectRelic(id) {
        this.currentRelicId = id;
        if (AdvancedFeatures) AdvancedFeatures.currentRelicId = id;

        document.querySelectorAll('.relic-item').forEach(item => {
            item.classList.toggle('active', parseInt(item.dataset.id) === id);
        });

        document.getElementById('relic-detail-panel').style.display = 'block';

        try {
            const detail = await API.getRelicDetail(id);
            this.currentRelicDetail = detail || this.generateMockDetail(id);
        } catch (e) {
            this.currentRelicDetail = this.generateMockDetail(id);
        }

        Stats.renderRelicDetail(this.currentRelicDetail);
        Stats.renderSensorStats(this.currentRelicDetail);

        const relicInfo = this.relics.find(r => r.id === id) || {};
        const fullData = Object.assign({}, relicInfo, this.currentRelicDetail);
        ThreeViewer.loadRelic(fullData);

        Contour.render(this.currentRelicDetail);
        Trends.loadData(id);
        CleaningSim.reset();
    },

    generateMockDetail(id) {
        const relic = this.relics.find(r => r.id === id) || { id, name: '文物 #' + id, location: '未知' };
        const sensors = [];
        const latestData = [];
        const usCount = id % 2 === 0 ? 4 : 3;
        const rtCount = 2;
        let sId = 1;

        for (let i = 0; i < usCount; i++) {
            sensors.push({
                id: (id - 1) * 5 + sId,
                relic_id: id,
                type: 'ultrasonic',
                model: 'US-300',
                position_x: i / Math.max(usCount - 1, 1),
                position_y: 0.5
            });
            sId++;
            const base = 0.3 + id * 0.15;
            const val = base + Math.random() * (2.5 + id * 0.1);
            latestData.push({
                sensor_id: (id - 1) * 5 + (i + 1),
                relic_id: id,
                latest_time: new Date(),
                latest_value: val,
                latest_unit: 'mm',
                latest_so2: 15 + id * 2 + Math.random() * 10,
                latest_humidity: 40 + Math.random() * 30,
                latest_temperature: 8 + Math.sin(id) * 8 + Math.random() * 3
            });
        }

        for (let i = 0; i < rtCount; i++) {
            sensors.push({
                id: 100 + (id - 1) * 3 + i + 1,
                relic_id: id,
                type: 'roughness',
                model: 'RT-200',
                position_x: 0.2 + i * 0.3,
                position_y: 0.8
            });
            latestData.push({
                sensor_id: 100 + (id - 1) * 3 + i + 1,
                relic_id: id,
                latest_time: new Date(),
                latest_value: 8 + id * 3 + Math.random() * 35,
                latest_unit: 'μm',
                latest_so2: 18 + Math.random() * 12,
                latest_humidity: 45 + Math.random() * 25,
                latest_temperature: 10 + Math.random() * 8
            });
        }

        const maxThickness = Math.max(...latestData.filter(d => d.latest_unit === 'mm').map(d => d.latest_value));
        const roughnessData = latestData.filter(d => d.latest_unit === 'μm');
        const avgRoughness = roughnessData.reduce((a, b) => a + b.latest_value, 0) / roughnessData.length;

        return Object.assign({}, relic, {
            sensors,
            latest_data: latestData,
            max_thickness: maxThickness,
            avg_roughness: avgRoughness,
            alert_count: maxThickness > 3 ? 2 : (maxThickness > 2 ? 1 : 0)
        });
    },

    async refreshData() {
        if (this.currentRelicId) {
            try {
                const detail = await API.getRelicDetail(this.currentRelicId);
                if (detail) {
                    this.currentRelicDetail = detail;
                    Stats.renderSensorStats(detail);
                    ThreeViewer.loadRelic(Object.assign({},
                        this.relics.find(r => r.id === this.currentRelicId) || {},
                        detail
                    ));
                }
            } catch (e) {}
        }
    },

    async refreshAlerts() {
        try {
            const stats = await API.getAlertStats();
            if (stats) Stats.renderAlertStats(stats);
            const alerts = await API.getAlerts(7, 20);
            if (alerts) Stats.renderAlerts(alerts);
        } catch (e) {}
    },

    handleWSAlert(data) {
        Stats.showToast(
            `🚨 告警 - ${this.getAlertTypeName(data.type)}`,
            `${this.getLevelName(data.level)}：当前值 ${data.value.toFixed(2)}，阈值 ${data.threshold}`,
            data.level
        );
        this.refreshAlerts();

        if (data.relic_id === this.currentRelicId) {
            this.refreshData();
        }
    },

    handleWSSensorData(data) {
        if (data.relic_id === this.currentRelicId) {
        }
    },

    handleWSStats(data) {
    },

    getAlertTypeName(type) {
        const names = { thickness: '结垢厚度', roughness: '表面粗糙度' };
        return names[type] || type;
    },

    getLevelName(level) {
        const names = { warning: '一般告警', critical: '严重告警' };
        return names[level] || level;
    }
};

document.addEventListener('DOMContentLoaded', () => App.init());
