let ws;
const listeners = [];
let connectionId;

export function initBrokerJavascript() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/ws`;
    console.log("using wsURL: " + wsUrl);
    if (!wsUrl || wsUrl.trim() === "") {
        console.log("unable to set up broker...no websocket url created");
        return;
    }

    ws = new WebSocket(wsUrl);

    ws.onopen = () => {
        console.log("WebSocket connected");
    };

    ws.onmessage = (e) => {
        const values = parseStrictTwoBrackets(e.data);
        const topic = values[0];
        const payload = values[1];

        if (topic === null || payload === null) {
            console.log(`invalid message [${e.data}]`);
            return;
        }

        if (topic === "broker_id") {
            console.log("hit broker_id");
            listeners.forEach(({ filter, callback }) => {
                if (!filter || filter.includes("broker_id")) {
                    callback(topic, payload);
                }
            });
            connectionId = payload;
            return;
        }

        const targetDiv = document.getElementById(topic);
        if (targetDiv) {
            targetDiv.textContent = payload;
        }

        listeners.forEach(({ filter, callback }) => {
            if (!filter || filter.includes(topic)) {
                callback(topic, payload);
            }
        });
    };

    return broker;
}

const broker = {
    send(topic, payload) {
        if (!ws || ws.readyState !== WebSocket.OPEN) {
            console.warn("WebSocket not connected");
            return;
        }
        if (topic === null || payload === null) {
            console.error(`topic [${topic}] | payload [${payload}] -> incorrect values`);
            return;
        }
        ws.send(`[${topic}][${payload}]`);
    },

    onMessage(topicFilter, callback) {
        // Allow topicFilter to be optional
        if (typeof topicFilter === "function") {
            callback = topicFilter;
            topicFilter = undefined;
        }

        if (typeof callback !== "function") {
            throw new TypeError("onMessage callback must be a function");
        }

        // Normalize filter to an array (or null if no filter)
        let filterArray = null;
        if (typeof topicFilter === "string") {
            filterArray = [topicFilter];
        } else if (Array.isArray(topicFilter)) {
            filterArray = topicFilter;
        }

        listeners.push({ filter: filterArray, callback });
    }
};

function parseStrictTwoBrackets(str) {
    if (str === null) {
        return [null, null];
    }

    const result = [];
    let current = "";
    let i = 0;

    for (let pair = 0; pair < 2; pair++) {
        if (str[i] !== "[") {
            result.push(null);
        } else {
            i++;
            current = "";
            while (i < str.length && str[i] !== "]") {
                current += str[i++];
            }
            result.push(current === "" ? null : current);
            if (str[i] === "]") i++;
        }
    }

    return result;
}

// --- test cases --- //
const tests = [
    ["[one][two]", ["one", "two"]],
    ["[][two]", [null, "two"]],
    ["[one][]", ["one", null]],
    ["[][]", [null, null]],
    ["[one][two][x]", ["one", "two"]],
    ["foo[one][two]", [null, null]],
    ["[one]junk[two]", ["one", null]],
    ["[onlyone]", ["onlyone", null]],
    ["", [null, null]],
    [null, [null, null]]
];

for (const [input, expected] of tests) {
    const tempInput = input === null ? "" : input;
    const output = parseStrictTwoBrackets(input);
    console.log(
        tempInput.padEnd(20),
        "=>",
        JSON.stringify(output),
        output.toString() === expected.toString() ? "✅" : `❌ expected ${JSON.stringify(expected)}`
    );
}
