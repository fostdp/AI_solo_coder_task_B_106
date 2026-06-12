const Stats = {
    renderAlertStats(data) {
        const container = document.getElementById('alert-stats');
        container.innerHTML = `
            <div class="alert-stat-item critical">
                <div class="num">${data.critical || 0}</div>
                <div class="lbl">严重告警</div>
            </div>
            <div class="alert-stat-item warning">
                <div class="num">${data.warning || 0}</div>
                <div class="lbl">一般告警</div>
            </div>
            <div class="alert-stat-item thickness">
                <div class="num">${data.thickness_alerts || 0}</div>
                <div class="lbl">厚度超标</div>
            </div>
            <div class="alert-stat-item roughness">
                <div class="num">${data.roughness_alerts || 0}</div>
                <div class="lbl">粗糙度超标</div>
            </div>
        `;
    },

    renderSensorStats(detail) {
        const container = document.getElementById('sensor-stats');
        const maxThickness = detail.max_thickness || 0;
        const avgRoughness = detail.avg_roughness || 0;
        const alertCount = detail.alert_count || 0;

        let so2 = 0, humidity = 0, temp = 0, count = 0;
        (detail.latest_data || []).forEach(d => {
            if (d.latest_so2) { so2 += d.latest_so2; }
            if (d.latest_humidity) { humidity += d.latest_humidity; }
            if (d.latest_temperature) { temp += d.latest_temperature; }
            if (d.latest_so2) count++;
        });
        if (count > 0) { so2 /= count; humidity /= count; temp /= count; }

        const thickClass = maxThickness > 3 ? 'danger' : (maxThickness > 2 ? 'warning' : '');
        const roughClass = avgRoughness > 50 ? 'danger' : (avgRoughness > 40 ? 'warning' : '');

        container.innerHTML = `
            <div class="stat-card">
                <div class="stat-label">最大结垢厚度</div>
                <div class="stat-value ${thickClass}">${maxThickness.toFixed(2)}</div>
                <div class="stat-unit">mm</div>
            </div>
            <div class="stat-card">
                <div class="stat-label">平均粗糙度</div>
                <div class="stat-value ${roughClass}">${avgRoughness.toFixed(1)}</div>
                <div class="stat-unit">μm</div>
            </div>
            <div class="stat-card">
                <div class="stat-label">SO₂ 浓度</div>
                <div class="stat-value">${so2.toFixed(1)}</div>
                <div class="stat-unit">ppb</div>
            </div>
            <div class="stat-card">
                <div class="stat-label">相对湿度</div>
                <div class="stat-value">${humidity.toFixed(0)}</div>
                <div class="stat-unit">%</div>
            </div>
            <div class="stat-card">
                <div class="stat-label">环境温度</div>
                <div class="stat-value">${temp.toFixed(1)}</div>
                <div class="stat-unit">°C</div>
            </div>
            <div class="stat-card">
                <div class="stat-label">近7天告警</div>
                <div class="stat-value ${alertCount > 0 ? 'warning' : ''}">${alertCount}</div>
                <div class="stat-unit">次</div>
            </div>
        `;
    },

    renderRelicDetail(detail) {
        const container = document.getElementById('relic-detail');
        container.innerHTML = `
            <div class="detail-row">
                <span class="detail-label">文物ID</span>
                <span class="detail-value">#${detail.id}</span>
            </div>
            <div class="detail-row">
                <span class="detail-label">名称</span>
                <span class="detail-value" style="color:#d4a017">${detail.name}</span>
            </div>
            <div class="detail-row">
                <span class="detail-label">位置</span>
                <span class="detail-value">${detail.location}</span>
            </div>
            <div class="detail-row">
                <span class="detail-label">传感器数</span>
                <span class="detail-value">${(detail.sensors || []).length}</span>
            </div>
            <div class="detail-row">
                <span class="detail-label">超声测点</span>
                <span class="detail-value">${(detail.sensors || []).filter(s => s.type === 'ultrasonic').length}</span>
            </div>
            <div class="detail-row">
                <span class="detail-label">粗糙测点</span>
                <span class="detail-value">${(detail.sensors || []).filter(s => s.type === 'roughness').length}</span>
            </div>
            <div class="detail-row">
                <span class="detail-label">最大厚度</span>
                <span class="detail-value ${detail.max_thickness > 3 ? 'critical' : (detail.max_thickness > 2 ? 'warning' : '')}">${(detail.max_thickness || 0).toFixed(3)} mm</span>
            </div>
            <div class="detail-row">
                <span class="detail-label">平均粗糙度</span>
                <span class="detail-value ${detail.avg_roughness > 50 ? 'critical' : (detail.avg_roughness > 40 ? 'warning' : '')}">${(detail.avg_roughness || 0).toFixed(1)} μm</span>
            </div>
        `;
    },

    renderAlerts(alerts) {
        const container = document.getElementById('alert-list');
        if (!alerts || alerts.length === 0) {
            container.innerHTML = '<div class="empty-state">暂无告警</div>';
            return;
        }

        const typeMap = {
            thickness: { name: '结垢厚度', icon: '📏' },
            roughness: { name: '表面粗糙度', icon: '🔬' }
        };

        container.innerHTML = alerts.slice(0, 20).map(a => {
            const t = typeMap[a.type] || { name: a.type, icon: '⚠️' };
            const time = new Date(a.created_at).toLocaleString('zh-CN');
            return `
                <div class="alert-item ${a.level}">
                    <div class="alert-header">
                        <span class="alert-type">${t.icon} ${t.name}</span>
                        <span class="alert-level ${a.level}">${a.level === 'critical' ? '严重' : '警告'}</span>
                    </div>
                    <div class="alert-value">当前值: <strong>${a.value.toFixed(2)}</strong> / 阈值: ${a.threshold}</div>
                    <div class="alert-time">文物#${a.relic_id} · ${time}</div>
                </div>
            `;
        }).join('');
    },

    renderCleaningLogs(records) {
        const container = document.getElementById('cleaning-log-list');
        if (!records || records.length === 0) {
            container.innerHTML = '<div class="empty-state">暂无清洗记录</div>';
            return;
        }

        container.innerHTML = records.slice(0, 10).map(r => {
            const time = new Date(r.created_at).toLocaleDateString('zh-CN');
            const error = r.actual_depth > 0 
                ? `误差: ${(Math.abs(r.actual_depth - r.predicted_depth) / r.predicted_depth * 100).toFixed(1)}%`
                : '';
            return `
                <div class="cleaning-log-item">
                    <div class="cleaning-log-header">
                        <span>文物#${r.relic_id} 区域${r.area_id}</span>
                        <span style="color:var(--accent-gold)">${time}</span>
                    </div>
                    <div class="cleaning-log-params">
                        <div>功率: ${r.laser_power}W</div>
                        <div>脉宽: ${r.pulse_duration}ns</div>
                        <div>速度: ${r.scan_speed}mm/s</div>
                        <div>预测: ${r.predicted_depth.toFixed(2)}mm</div>
                    </div>
                    <div style="margin-top:4px;color:var(--secondary);font-size:11px;">${error}</div>
                </div>
            `;
        }).join('');
    },

    showToast(title, message, type = 'info') {
        const container = document.getElementById('toast-container');
        const toast = document.createElement('div');
        toast.className = `toast ${type}`;
        toast.innerHTML = `
            <div class="toast-title">${title}</div>
            <div class="toast-message">${message}</div>
        `;
        container.appendChild(toast);
        setTimeout(() => toast.remove(), 4000);
    }
};
