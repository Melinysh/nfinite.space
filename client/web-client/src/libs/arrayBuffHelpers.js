
function file2ab(f) {
  return new Promise((res, rej) => {
    const reader = new FileReader();
    reader.onload = (f => {
      return e => {
        const fileArrayBuffer = e.target.result;

        res(fileArrayBuffer);
      };
    })(f);

    reader.readAsArrayBuffer(f);
  })
}

export {file2ab };
