# Microservice Markdown Magic

This is a playground project to showcase some features of Platform.sh.

The "core" is `controller_microservice`, an app written in Golang to parse Markdown input and render it into HTML.
It leverages https://github.com/gomarkdown/markdown, using it for its core functions with some overrides.

Communication with other microservices is enabled by the `PLATFORM_ROUTES` variable, populated through [routes.yaml](.platform/routes.yaml).
On each request, the controller will look at `PLATFORM_ROUTES` and `GET` the `/discover` path of each route found there. That request should return something like

```json
{
	"name": "pygments",
	"type": "*ast.CodeBlock",
	"attrs": {
		"language": "Info"
	},
	"flags": {
		"composable": false
	},
}
```

`name` is self-explanatory. `type` is the type of Markdown element your service ought to operate on; see [gomarkdown's AST documentation](https://godoc.org/github.com/gomarkdown/markdown/ast) for a list.
The three example microservices implement `*ast.CodeBlock`, `*ast.Text` and `*ast.Heading`.

`attrs` represents the attributes of the Markdown element you would like to pass to this microservise. In the example above, this means the contents of the `Info` field of `CodeBlock` objects will be passed to the microservice in the `language` field. The [gomarkdown AST docs](https://godoc.org/github.com/gomarkdown/markdown/ast) have all the details on what is available.

`flags` are static flags pertaining to this microservice. If absent, they are treated as false. Currently only `composable` is implemented; it means that rendering output is safe to feed into the next microservice. Usually because it's just text. Services that produce html shouldn't be `composable`, those that output only text should be (see the "redacted" service).
