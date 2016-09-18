export default function WebSocketPlus(server) {
  const ws = new WebSocket(server);

  this.onOpen = () => {};
  this.onClose = () => {};
  this.onMessage = () => {};

  this.sendJSON = json => {
    ws.send(JSON.stringify(json));
    console.log("Sending JSON string:", JSON.stringify(json));
  }
  this.sendBuffer = ab => {
    ws.send(ab)
    console.log("Sending Buffer:", ab)
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
