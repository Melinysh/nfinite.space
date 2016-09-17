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
function handleDownloadRequest (fileName){
  console.log(fileName)
}

function loadTable(elements){
  var table = document.getElementById("fileList");
  for (var x = 0;x<elements.length;x++){
    var row = table.insertRow(x+1)
    var fileNameCell = row.insertCell(0)
    fileNameCell.innerHTML = elements[x]
    var downloadCell = row.insertCell(1)

    var button = document.createElement('button')
    button.setAttribute("associatedFileName", fileNameCell.innerHTML)
    button.innerHTML = "Download"

    button.onclick = function(){

      handleDownloadRequest(this.getAttribute("associatedFileName"));
    }


    downloadCell.appendChild(button)
  }

}


const FILES = {};
const fileArray = {};
function WebSocketPlus(server) {
    const ws = new WebSocket(server);

    this.onOpen = () => {};
    this.onClose = () => {};

    const state = {
        expectingFileList:true,
        expectingBlob: false,

        blobProm: null,
        jsonProm: null
    }

    this.onMessage = evt => {
         //console.log(evt.data, evt);

        if (state.expectingBlob) {
            console.log("Got a Blob")

            state.blobProm = Promise.resolve(evt.data);
        } else {
          if (state.expectingFileList){
            console.log("Got list of files")
            const fileListProm = blob2buf(evt.data)
            Promise.all([fileListProm]).then(vals => {
              this.fileArray = vals[0]["files"]
              loadTable(this.fileArray);
            })

          }
          else{
            console.log("Got some JSON")

            state.jsonProm = blob2buf(evt.data);

            state.expectingBlob = true;
          }
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

    ws.onopen = () => {
        console.log("Connection Established");
        this.onOpen();
    }
    ws.onclose = () => {
        console.log("Connection Terminated");
        this.onClose();
    }

    ws.onmessage = evt => this.onMessage(evt);
}

const ws = new WebSocketPlus("ws://10.20.170.46:8080/websockets")

ws.onOpen = () => {
    ws.sendJSON({
        type: "registration",
        userMeta: {
            name: "dickbutt",
            pass: "poopbutt"
        }
    })
}

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
