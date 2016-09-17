function ab2str(buf) {
    return String.fromCharCode.apply(null, new Uint8Array(buf));
}

function blob2buf(blob) {
    return new Promise((res, rej) => {
        const reader = new FileReader();
        reader.onload = () => {
            const msg = JSON.parse(ab2str(reader.result));
            res(msg)
        };
        reader.readAsArrayBuffer(blob);
    })
}


const FILES = {};

function WebSocketPlus(server) {
    const ws = new WebSocket(server);

    this.onopen = () => {
        console.log("Connection Established")
    }

    this.onclose = () => {
        console.log("Connection Terminated")
    }

    const state = {
        expectingBlob: false,

        blobProm: null,
        jsonProm: null
    }

    this.onmessage = evt => {
        // console.log(evt.data, evt);

        if (state.expectingBlob) {
            console.log("Got a Blob")

            state.blobProm = Promise.resolve(evt.data);
        } else {
            console.log("Got some JSON")

            state.jsonProm = blob2buf(evt.data);

            state.expectingBlob = true;
        }

        Promise.all([
            state.blobProm,
            state.jsonProm
        ]).then(vals => {
            if (vals.some(x => x === null)) return;

            FILES[vals[1].fileMeta.name] = vals[0];

            console.log("Stored blob in FILES");
            console.log(FILES);

            state.blobProm = null;
            state.jsonProm = null;

            state.expectingBlob = false;
        })

    }

    this.sendArrayBuffer = ab => ws.send(ab);
    this.sendJSON = json => {
        ws.send(JSON.stringify(json));
        console.log("sending", JSON.stringify(json));

    }

    ws.onopen = () => this.onopen();
    ws.onmessage = evt => this.onmessage(evt);
    ws.onclose = () => this.onclose();
}

const ws = new WebSocketPlus("ws://10.20.170.46:8080/websockets")

function handleFileSelect(evt) {
    const files = evt.target.files; // FileList object

    // files is a FileList of File objects. List some properties.
    const output = [];
    for (let i = 0, f; f = files[i]; i++) {

        const reader = new FileReader();
        reader.onload = (f => {
            return e => {
                const fileArrayBuffer = e.target.result;

                ws.sendJSON({
                    type: "file",
                    fileMeta: {
                        "name": escape(f.name),
                        "dateModified": f.lastModifiedDate.getTime().toString()
                    }
                });

                ws.sendArrayBuffer(fileArrayBuffer);
            };
        })(f);

        reader.readAsArrayBuffer(f);
    }
}

document.getElementById('files').addEventListener('change', handleFileSelect, false);
