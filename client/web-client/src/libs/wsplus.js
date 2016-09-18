export default function WebSocketPlus(server) {
  const ws = new WebSocket(server);

  this.onOpen = () => {};
  this.onClose = () => {};
  this.onMessage = () => {};

  this.sendJSON = json => {
    ws.send(JSON.stringify(json));
    console.log("Sending JSON string:", JSON.stringify(json));

    document.querySelector(".INDICATOR_UP").style.opacity = 1;
    setTimeout(() => {
      document.querySelector(".INDICATOR_UP").style.opacity = 0.5
      setTimeout(() => {
        document.querySelector(".INDICATOR_UP").style.opacity = 0
      }, 500)
    }, 500)
  }
  this.sendBuffer = ab => {
    ws.send(ab)
    console.log("Sending Buffer:", ab)

    document.querySelector(".INDICATOR_UP").style.opacity = 1;
    setTimeout(() => {
      document.querySelector(".INDICATOR_UP").style.opacity = 0.5
      setTimeout(() => {
        document.querySelector(".INDICATOR_UP").style.opacity = 0
      }, 500)
    }, 500)
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

    document.querySelector(".INDICATOR_DOWN").style.opacity = 1;
    setTimeout(() => {
      document.querySelector(".INDICATOR_DOWN").style.opacity = 0.5
      setTimeout(() => {
        document.querySelector(".INDICATOR_DOWN").style.opacity = 0
      }, 500)
    }, 500)

    this.onMessage(evt);
  }

}
