import React from 'react';

import FileList from './FileList';

const App = props => {
  /* beautify preserve:start */
  return (
    <div className="wrapper">
      <h2 className="tableHeader">Upload your file</h2>

      <input className="selectFile"
             onChange={props.handlers.handleFileUpload}
             type="file"
             id="files"
             name="files[]"
             multiple />
      
      <h2 className="tableHeader">Your Files</h2>

      <FileList files={props.fileArray} 
                downloadHandler={props.handlers.handleDownloadRequest}/>

      {/* Some status indicators for the demo */}
      <div className="INDICATOR INDICATOR_UP"></div>
      <div className="INDICATOR INDICATOR_DOWN"></div>
    </div>
  )
  /* beautify preserve:end */
}

export default App;
