import React, { Component } from 'react';

import App from './App';

import WebSocketPlus from '../libs/wsplus';

// HELPERS

function file2ab(f) {
  return new Promise((res, rej) => {
    const reader = new FileReader();
    reader.onload = (f => {
      return e => {
        const fileArrayBuffer = e.target.result;

        res(fileArrayBuffer);
      };
    })(f);

    reader.readAsArrayBuffer(f);
  })
}

function saveFileFromArrayBuffer(fullName, buf) {
  const blob = new Blob([buf]);

  const link = document.createElement('a');
  link.href = window.URL.createObjectURL(blob);
  link.download = fullName;

  link.click();
}

// MAIN APP

const PART_STORE = {};

class AppContainer extends Component {
  state = {
    fileArray: []
  }

  _blobMetaJSON = {}

  componentDidMount = () => {
    this._ws = new WebSocketPlus("ws://54.197.38.216:8080/websockets");
    this._ws.onOpen = () => {
      this._ws.sendJSON({
        type: "registration",
        userMeta: {
          name: window.username ? window.username : "DEFAULT",
          pass: window.password ? window.password : "DEFAULT"
        }
      })
    }

    this._ws.onMessage = evt => {
      const data = evt.data;

      console.log("got msg event:", evt);

      if (typeof data === "string") {
        const json = JSON.parse(data);
        console.log("Got JSON!", JSON.parse(data));

        switch (json.type) {
          /* User comms */
          case "fileList":
            console.log("Got list of files",  json["files"].map(x => x.fileMeta))

            this.setState({
              fileArray: json["files"].map(x => x.fileMeta)
            })

            break;
          case "response":
            console.log("Got File meta, next message must be the file blob");

            this._blobMetaJSON = json
            break;

            /* User-as-storage comms */
          case "part":
            console.log("Got Part meta, next message must be a part blob")

            this._blobMetaJSON = json
            break;
          case "request":
            console.log("Got part request")

            this.$handlePartRequest(json["fileMeta"]["name"])
            break;

        }
      } else if (data.constructor.name === "Blob") {
        console.log("Got Data Blob!", data);

        file2ab(data)
          .then(ab => {
            if (this._blobMetaJSON.type === "response") {
              console.log("Received file arraybuffer: ", ab);

              saveFileFromArrayBuffer(this._blobMetaJSON["fileMeta"]["name"], ab);

            } else if (this._blobMetaJSON.type === "part") {
              console.log("Storing part in PARTS");

              PART_STORE[this._blobMetaJSON["fileMeta"]["name"]] = ab;

              console.log(PART_STORE);
            }
          })
      }

    }
  }

  $handlePartRequest = fileName => {
    console.log("PARTS " + PART_STORE + " Finding part " + fileName)

    if (PART_STORE[fileName] !== null) {
      console.log("Do we have it? " + PART_STORE[fileName])

      this._ws.sendBuffer(PART_STORE[fileName])
    } else {
      console.log("No such part exists")
    }
  }

  handleDownloadRequest = (fileName) => {
    console.log("Requested:", fileName);

    this._ws.sendJSON({
      type: "request",
      "fileMeta": {
        name: fileName,
        dateModified: ""
      }
    })
  }

  handleFileUpload = evt => {
    const files = evt.target.files; // FileList object

    Object.keys(files)
      .map(key => files[key])
      .forEach(f => {
        file2ab(f).then(ab => {
          console.log(escape(f.name));

          this._ws.sendJSON({
            type: "file",
            fileMeta: {
              "name": escape(f.name),
              "dateModified": f.lastModifiedDate.getTime().toString()
            }
          });
          this._ws.sendBuffer(ab);

          // update file-list

          const newFileArray = this.state.fileArray;
          newFileArray.push({
            name:escape(f.name),
            lastModified: (f.lastModifiedDate.getTime() / 1000).toString()
          })

          this.setState({
            fileArray: newFileArray
          })
        })
      })
  }


  _handlers = {
    handleFileUpload: this.handleFileUpload.bind(this),
    handleDownloadRequest: this.handleDownloadRequest.bind(this),
  }

  /* beautify preserve:start */
  render() {
    return (
      <App {...this.state}
           handlers={this._handlers}/>
    )
  }
  /* beautify preserve:end */
}

export default AppContainer;
