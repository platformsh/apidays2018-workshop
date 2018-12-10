import base64
import json
import os
import traceback
import uuid
import sys

import flask
import gevent.pywsgi

import pygments
import pygments.formatters
import pygments.lexers
import pygments.util

app = flask.Flask(__name__)

relationships = json.loads(base64.b64decode(os.environ["PLATFORM_RELATIONSHIPS"]).decode())


@app.route('/', methods=["POST"])
def root():
    language = flask.request.form.get("language", "")
    text = flask.request.form.get("text", "")
    app.logger.warning(language)
    app.logger.warning(text)

    try:
        lexer = pygments.lexers.get_lexer_by_name(language)
    except pygments.util.ClassNotFound:
        lexer = pygments.lexers.guess_lexer(text)

    app.logger.warning(lexer)

    output = pygments.highlight(text, lexer, pygments.formatters.HtmlFormatter(noclasses=True))

    return output


@app.route('/discover', methods=["GET"])
def discover():
    data = {"name": "pygments", "type": "*ast.CodeBlock", "attrs": {"language": "Info"}}
    return flask.jsonify(data)


if __name__ == "__main__":
    http_server = gevent.pywsgi.WSGIServer(('127.0.0.1', int(os.environ["PORT"])), app)
    http_server.serve_forever()
