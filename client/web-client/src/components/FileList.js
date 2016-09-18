import React from 'react';

const FileList = props => {
  console.log(props)

  // file.lastModified

  let i = 0;
  const Files = props.files.map(file => {
    const fulldate = new Date(file.lastModified*1000).toString();
    const dateString = /(.*? .*? .*? .*?) .*/.exec(fulldate)[1]

    /* beautify preserve:start */
    return (
      <a key={i++} onClick={() => props.downloadHandler(file.name)}>
        <div className="file">
          <div className="file__name">
            {file.name}
          </div>
          <div className="file__download">
              {dateString}
          </div>
        </div>
      </a>
    )
  /* beautify preserve:end */
  })

  /* beautify preserve:start */
  return (
    <div className="fileListWrapper">
      <div className="file fileTitle">
        <div className="file__name">
          File Name
        </div>
        <div className="file__download">
          Date Modified
        </div>
      </div>
      {Files}
    </div>
  )
  /* beautify preserve:end */
}

export default FileList;
