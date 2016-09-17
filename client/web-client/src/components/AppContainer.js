import React, { Component } from 'react';

import App from './App';

import WebSocketPlus from '../libs/wsplus';

import * as convert from '../libs/arrayBuffHelpers';

class AppContainer extends Component {
  state = {
    fileArray: []
  }

  PARTS = {};
  safeKeepingJSON = {}

    componentDidMount = () => {
      this._ws = new WebSocketPlus("ws://54.197.38.216:8080/websockets");
      this._ws.onOpen = () => {
        this._ws.sendJSON({
          type: "registration",
          userMeta: {
            name: "dickbutt",
            pass: "poopbutt"
          }
        })
      }

      this._ws.onMessage = evt => {
        const data = evt.data;

        console.log(evt);

        if (typeof data === "string") {
          const json = JSON.parse(data);
          console.log("Got JSON!", JSON.parse(data));

          if (json.type === "fileList"){
            console.log("Got list of files")
            const fileList = JSON.parse(evt.data)
              this.loadTable(fileList["files"]);
              this.setState({fileArray:fileList["files"]})
          }

          if (json.type === "part") {
            console.log("PART RECIEVED" + json)
            this.safeKeepingJSON = json
          }

          if (json.type === "request"){
            console.log("REQUEST RECIEVED")
            this.handlePartRequest(this._ws,json["fileMeta"]["name"])
          }

          if (json.type == "response"){
            console.log("RESPONSE RECEIVED");
            this.safeKeepingJSON = json

          }

        } else if (data.constructor.name === "Blob") {
          console.log("Got Data Blob!", data);
          if (this.safeKeepingJSON.type === "response" ){
            //this means i got a file!
            convert.file2ab(data).then(ab => {
                console.log("Received file: " + ab)

                this.saveByteArray(this.safeKeepingJSON["fileMeta"]["name"],ab)
                this.safeKeepingJSON = null
            })

          }
          else if(this.safeKeepingJSON.type === "part"){
            convert.file2ab(data).then(ab => {
                console.log("AB: " +ab)
                this.PARTS[this.safeKeepingJSON["fileMeta"]["name"]] = ab;

                console.log("Stored blob in PARTS");
                console.log(this.PARTS);
            })

          }

        }

      }
    }

   saveByteArray = (fullName, byte) => {
      var blob = new Blob([byte]);
      var link = document.createElement('a');
      link.href = window.URL.createObjectURL(blob);
      var timeNow = new Date();
      var month = timeNow.getMonth() + 1;
      var fileName = fullName;
      link.download = fileName;
      link.click();
  }
  handlePartRequest = (socket,fileName) => {

    console.log("PARTS " + this.PARTS + " Finding part " + fileName)
    if (this.PARTS[fileName] !== null){
      console.log("Do we have it? " + this.PARTS[fileName])
      socket.sendBuffer(this.PARTS[fileName])
      this.safeKeepingJSON = null
    }
    else{
      console.log("No such part exists")
    }
  }

   handleDownloadRequest = (fileName) => {
    this._ws.sendJSON({
        type: "request",
        "fileMeta":{
          name:"profile5.jpg",
          dateModified: ""
        }
    })
    console.log(fileName)
  }

  loadTable = (elements) => {
      var table = document.getElementById("fileList");
      for (var x = 0;x<elements.length;x++){
        var row = table.insertRow(x+1)
        var fileNameCell = row.insertCell(0)
        fileNameCell.innerHTML = elements[x]
        var downloadCell = row.insertCell(1)

        var button = document.createElement('button')

        button.setAttribute("associatedFileName", fileNameCell.innerHTML)
        button.innerHTML = "Download"
        var self = this
        button.onclick = function() {
          self.handleDownloadRequest(this.getAttribute("associatedFileName"));
        }
        downloadCell.appendChild(button)
  }
}
handleFileUpload = evt => {
    const files = evt.target.files; // FileList object

    Object.keys(files)
      .map(key => files[key])
      .forEach(f => {
        convert.file2ab(f).then(ab => {
          console.log(escape(f.name));

          this._ws.sendJSON({
            type: "file",
            fileMeta: {
              "name": escape(f.name),
              "dateModified": f.lastModifiedDate.getTime().toString()
            }
          });

          this._ws.sendArrayBuffer(ab);
        })
      })
  }


_handlers = {
   handleFileUpload: this.handleFileUpload.bind(this),
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
