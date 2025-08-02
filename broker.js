let ws;
const listeners = [];

export function initBrokerJavascript(wsUrl = null) {
    // Use relative WebSocket URL if not provided
    if (!wsUrl) {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        wsUrl = `${protocol}//${window.location.host}/ws`;
    }
    
    console.log("using wsURL: " + wsUrl);
    
    ws = new WebSocket(wsUrl);

    ws.onopen = () => {
        console.log("WebSocket connected");
    };

    ws.onmessage = (e) => {
        console.log("from server:", e.data);
        const matches = e.data.match(/\[([^\]]+)\]/g);

        if (matches && matches.length >= 2) {
            const fields = matches.map((s) => s.slice(1, -1));
            const topic = fields[0];
            const value = fields[1];

            const targetDiv = document.getElementById(topic);
            if (targetDiv) {
                targetDiv.textContent = value;
            }

            listeners.forEach((cb) => cb(topic, value));
        } else {
            console.warn("Unexpected message format:", e.data);
        }
    };

    return broker;
}

const broker = {
    send(topic, value) {
        if (!ws || ws.readyState !== WebSocket.OPEN) {
            console.warn("WebSocket not connected");
            return;
        }
        ws.send(`[${topic}][${value}]`);
    },
    onMessage(callback) {
        listeners.push(callback);
    }
};
