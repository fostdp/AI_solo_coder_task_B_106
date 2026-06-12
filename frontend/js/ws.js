const WS = {
    socket: null,
    reconnectAttempts: 0,
    maxReconnectAttempts: 10,
    listeners: {},

    connect() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const host = '127.0.0.1:8080';
        const url = `${protocol}//${host}/ws`;

        try {
            this.socket = new WebSocket(url);
        } catch (e) {
            this.updateStatus(false);
            this.scheduleReconnect();
            return;
        }

        this.socket.onopen = () => {
            this.reconnectAttempts = 0;
            this.updateStatus(true);
            console.log('[WS] Connected');
        };

        this.socket.onclose = () => {
            this.updateStatus(false);
            console.log('[WS] Disconnected');
            this.scheduleReconnect();
        };

        this.socket.onerror = (e) => {
            console.error('[WS] Error:', e);
            this.updateStatus(false);
        };

        this.socket.onmessage = (event) => {
            try {
                const msg = JSON.parse(event.data);
                this.emit(msg.type, msg.data);
            } catch (e) {
                console.error('[WS] Parse error:', e);
            }
        };
    },

    scheduleReconnect() {
        if (this.reconnectAttempts >= this.maxReconnectAttempts) {
            console.error('[WS] Max reconnect attempts reached');
            return;
        }
        this.reconnectAttempts++;
        const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts), 30000);
        console.log(`[WS] Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts})`);
        setTimeout(() => this.connect(), delay);
    },

    updateStatus(connected) {
        const dot = document.getElementById('ws-status');
        const text = document.getElementById('ws-status-text');
        if (connected) {
            dot.className = 'status-dot connected';
            text.textContent = '已连接';
            text.style.color = 'var(--secondary)';
        } else {
            dot.className = 'status-dot';
            text.textContent = '未连接';
            text.style.color = 'var(--text-secondary)';
        }
    },

    on(event, callback) {
        if (!this.listeners[event]) {
            this.listeners[event] = [];
        }
        this.listeners[event].push(callback);
    },

    emit(event, data) {
        if (this.listeners[event]) {
            this.listeners[event].forEach(cb => cb(data));
        }
    },

    send(data) {
        if (this.socket && this.socket.readyState === WebSocket.OPEN) {
            this.socket.send(JSON.stringify(data));
        }
    }
};
