import React, { Component } from 'react';
import WebSocketPlus from '../libs/wsplus';
class Login extends Component {
  state = {
    username: null,
    password: null
  }
  componentDidMount = () => {

  }
  handleLogin = () =>{

    this._ws = new WebSocketPlus("ws://54.197.38.216:8080/websockets");
    this._ws.onOpen = () => {
      this._ws.sendJSON({
        type: "registration",
        userMeta: {
          name: this.state.username,
          pass: this.state.password
        }
      })

      window.location.href = '/#/dashboard'
    }
  }
  handleUserChange = (e) =>{
    this.setState({username:e.target.value});
  }
  handlePassChange = (e) =>{
    this.setState({password:e.target.value});
  }
  /* beautify preserve:start */
  render() {

    return (
      <div>
        <h1 className="tableHeader">Login</h1>
        <div className="container">
      <h3  className="formTitle"><b>Username</b></h3>
      <input onChange={this.handleUserChange} type="text" placeholder="Enter Username" name="uname" required />

      <h3 className = "formTitle"><b>Password</b></h3>
      <input onChange={this.handlePassChange}  type="password" placeholder="Enter Password" name="psw" required />

      <button onClick={this.handleLogin} type="submit">Login</button>

    </div>
      </div>
    )
  }
  /* beautify preserve:end */
}
export default Login;
