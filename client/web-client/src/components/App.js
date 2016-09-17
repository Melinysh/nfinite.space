import React from 'react';

const App=props => {
  return (
    <div className="wrapper">
      <input onChange={props.handlers.handleFileUpload} type="file" id="files" name="files[]" multiple />
      <table id="fileList">
        <tbody>
    <tr>
      <th>File Name</th>
      <th>Download</th>
    </tr>
      </tbody>
  </table>
    </div>
  )
}

export default App;
