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



      window.username = this.state.username;
      window.password = this.state.password;
      this.props.history.push('/dashboard');
    
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

      <button onClick={this.handleLogin} type="submit" className="button1">Login</button>

    </div>
      </div>
    )
  }
  /* beautify preserve:end */
}
export default Login;
