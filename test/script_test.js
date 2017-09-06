
var response = getOrFail(context.Execute({URL: "ssh://127.0.0.1:22/etc"}, {
    Executions: [
        {Command: "cat hosts"}
    ]
}));


var stdout = response.Stdout[0];
var lines = stdout.split("\r\n")
var state = context.State();
for (var i = 0; i < lines.length; i++) {
    var line = lines[i];
    if (line.substring(0, 1) === '#') continue;
    var fragments = line.split("\t")
    state[fragments[0]] = fragments[1]
}



var response = getOrFail(context.Execute({URL: "ssh://127.0.0.1:22/etc"}, {
    Executions: [
        {
            Command: "echo $PATH"
        }
    ]
}));
var stdout = response.Stdout[0];
console.log(stdout)
var result = "ok";
result;


