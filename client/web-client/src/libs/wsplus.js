// import * as convert from "./arrayBufHelpers";
export default function WebSocketPlus(server) {
  const ws = new WebSocket(server);
  this.onOpen = () => {};
  this.onClose = () => {};
  this.onMessage = () => {};
  this.sendArrayBuffer = ab => ws.send(ab);
  this.sendJSON = json => {
    ws.send(JSON.stringify(json));
    console.log("Sending", JSON.stringify(json));
  }
  this.sendBuffer = data => {
    ws.send(data)
    console.log("Sending Buffer " + data)

  }
  ws.onopen = () => {
    console.log("Connection Established");
    this.onOpen();
  }
  ws.onclose = () => {
    console.log("Connection Terminated");
    this.onClose();
  }
  ws.onmessage = (evt) => {
    console.log("Recieved Message:", evt.data);
    this.onMessage(evt);
  }

}
