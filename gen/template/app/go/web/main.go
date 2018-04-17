package main

import (
	"io"
	"net/http"
)


func hello(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, index)
}

func main() {
	http.HandleFunc("/", hello)
	http.ListenAndServe(":8080", nil)
}



var index = `
	<!DOCTYPE html>
<html>
<head>
  <title>Hello world</title>
</head>
<script type="text/javascript">
  
(function(){

  var run = function() {
    var name = document.querySelector('#name')
    var output = document.querySelector('#output');
    output.innerHTML = 'Hello ' + name.value;
  }

  var onLoad = function() {
	var runButton = document.querySelector('#run');
  	console.log(runButton);
    runButton.addEventListener('click', run); 
  }

  window.addEventListener('load', onLoad);  

})()

</script>
<body>

<p>
  <label for="name">Name</label>
  <input type="text" name="name" id="name" />
</p>
<p>
  <input type="button" value="run" id="run" />
</p>

<div id="output">Hello ***</div>

</body>
</html>
`
