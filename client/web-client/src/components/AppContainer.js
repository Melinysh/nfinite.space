import React, { Component } from 'react';

import App from './App';

class AppContainer extends Component {
  state = {
    testing: "test"
  }

  render() {
    return (
      <App {...this.state}/>
    )
  }
}

export default AppContainer;