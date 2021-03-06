// Load the http module to create an http server.
var http = require('http');

// Load the Platform.sh configuration.

try {
        var config = require("platformsh").config();
    } catch(error) {
        console.error(error);
        var config = {}
}

// load the querystring module to parse POST data to JSON
const { parse } = require('querystring');

// load the text-to-svg extension that does the actual work
const TextToSVG = require('text-to-svg');
const textToSVG = TextToSVG.loadSync('./fonts/Bangers-Regular.otf');


// configure SVG options
var attributes = {fill: 'midnightblue', stroke: 'darkslateblue'};
var options = {x: 0, y: 0, fontSize: 72, anchor: 'top', attributes: attributes};

const heading_font_size = {
    "1": 72,
    "2": 48,
    "3": 36,
    "4": 28,
    "5": 24,
    "6": 20
}

var server = http.createServer(function (request, response) {
    
    if (request.method === "POST") {
        var body = "";
        request.on("data", function (chunk) {
            body += chunk;
        });

        request.on("end", function(){
            parsed = parse(body)
            if ("text" in parsed) {
                text = parsed.text
            } else {
                text =""
            }
            if ("heading_level" in parsed) {
                options.fontSize = heading_font_size[parsed.heading_level]
            }
            svg = textToSVG.getSVG(text, options)

            response.writeHead(200, { "Content-Type": "text/html" });
            console.log("Received text to render as SVG: " + body);
            response.end(svg);
        });
	}
	else {
        console.log("Received a discover request")
        if (request.url == '/discover') {
            response.writeHead(200, {"Content-Type": "application/json"})
            data = {
                "name": "svg",
                "type": "*ast.Heading",
                "attrs": {"heading_level": "Level"}
            }
            response.end(JSON.stringify(data))
        }
    }
});

server.listen(config.port||8080);