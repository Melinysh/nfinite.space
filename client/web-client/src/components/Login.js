import React, { Component } from 'react';

class Login extends Component {
  state = {}

  handleLogin = () => {
    this.props.history.push('/dashboard');
  }

  handleUserChange = e => {
    window.username = e.target.value;
  }

  handlePassChange = e => {
    window.password = e.target.value;
  }

  render() {
    /* beautify preserve:start */
    return (
      <div>
        <h1 className="tableHeader">Login</h1>

        <div className="formContainer">
          <h3 className="formTitle"><b>Username</b></h3>
          <input onChange={this.handleUserChange}
                 type="text"
                 placeholder="Enter Username"
                 name="uname"
                 required />

          <h3 className="formTitle"><b>Password</b></h3>
          <input onChange={this.handlePassChange}
                 type="password"
                 placeholder="Enter Password"
                 name="psw"
                 required />

          <button onClick={this.handleLogin}
                  type="submit"
                  className="loginButton">
            Login
          </button>

        </div>
      </div>
    )
  /* beautify preserve:end */
  }
}
export default Login;
