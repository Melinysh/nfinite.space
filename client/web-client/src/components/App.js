import React from 'react';
import { Router, Route, hashHistory } from 'react-router'
const App=props => {
  return (

    <div className="wrapper">
      <h2 className="tableHeader">Upload your file</h2>
      <input className="selectFile" onChange={props.handlers.handleFileUpload} type="file" id="files" name="files[]" multiple />
      <h2 className = "tableHeader">Your Files</h2>
      <table id="fileList">
        <tbody>
    <tr>
      <th>File Name</th>
      <th>Last Modified</th>
      <th>Download</th>
    </tr>
      </tbody>
  </table>
    </div>
  )
}

export default App;
