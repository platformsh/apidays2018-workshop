package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/html"
	psh "github.com/platformsh/gohelper"
	"gopkg.in/oleiade/reflections.v1"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"sort"
	"strings"
)

type Microservice struct {
	Name  string
	Route string
	Type  string
	Flags map[string]bool   // flags such as "composable" to pass to renderer
	Attrs map[string]string // attributes to get from node and pass to service if possible
}

func main() {

	// The psh library provides Platform.sh environment information mapped to Go structs.
	p, err := psh.NewPlatformInfo()

	if err != nil {
		// This means we're not running on Platform.sh!
		// In practice you would want to fall back to another way to define
		// configuration information, say for your local development environment.
		fmt.Println(err)
		panic("Not in a Platform.sh Environment.")

	}

	fmt.Println("Yay, found Platform.sh info")

	// Set up an extremely simple web server response.
	http.HandleFunc("/", handleFunc)

	// The port to listen on is defined by Platform.sh.
	log.Fatal(http.ListenAndServe(":"+p.Port, nil))
}

// checkErr is a simple wrapper for panicking on error.
// It likely should not be used in a real application.
func checkErr(err error) {
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
}

func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
}

func handleFunc(w http.ResponseWriter, r *http.Request) {
	var microservices []Microservice

	enableCors(&w)

	microservices, err := discoverServices()
	checkErr(err)
	fmt.Println("found microservices: ")
	fmt.Println(microservices)
	fmt.Println("sorting microservices: ")
	sort.Slice(microservices, func(i, j int) bool {
		if composable, ok := microservices[i].Flags["composable"]; ok {
			return composable
		}
		return false
	})
	fmt.Println(microservices)

	// this functions overrides default rendering in certain cases
	renderHook := func(w io.Writer, node ast.Node, entering bool) (ast.WalkStatus, bool) {

		renderStatus := false
		content := getContent(node)
		// skip all nodes that are not of types we already consider
		for _, microservice := range microservices {
			if match, parent := parentMatch(node, microservice.Type); match {
				params := map[string]string{}

				for k, v := range microservice.Attrs {
					value, e := reflections.GetField(parent, v)
					checkErr(e)
					// whether it is a string, a byte array, or a number, make it a string
					var s string
					rt := reflect.TypeOf(value)
					switch rt.Kind() {
					case reflect.Array:
						s = fmt.Sprintf("%s", value)
					case reflect.Slice:
						s = fmt.Sprintf("%s", value)
					case reflect.String:
						s = fmt.Sprintf("%s", value)
					default:
						s = fmt.Sprintf("%v", value)
					}
					params[k] = s
				}


				response := postToMicroservice(microservice.Route, content, params)
				s, err := ioutil.ReadAll(response.Body)
				checkErr(err)

				renderStatus = true
				content = string(s)
				if composable, ok := microservice.Flags["composable"]; ok {
					if composable != true {
						io.WriteString(w, content)
						return ast.GoToNext, renderStatus
					}
				} else {
					io.WriteString(w, content)
					return ast.GoToNext, renderStatus
				}
			}
		}
		if renderStatus { // we need to render something, because we ended on a "composable" microservice
			io.WriteString(w, content)
		}
		// this means we render the node as normal
		return ast.GoToNext, renderStatus
	}

	r.ParseForm()
	fmt.Println(r)

	if r.PostForm != nil {
		text := r.PostForm.Get("text")
		opts := html.RendererOptions{
			Flags:          html.CommonFlags,
			RenderNodeHook: renderHook,
		}
		renderer := html.NewRenderer(opts)

		fmt.Println("Received request, rendering...")
		md := []byte(text)
		html := markdown.ToHTML(md, nil, renderer)
		fmt.Fprintln(w, string(html[:]))
	}
}

// search if a parent of the node matches the type of the microservice
func parentMatchRecur(node ast.Node, typeName string) (bool, ast.Node) {
	if reflect.TypeOf(node).String() == typeName {
		return true, node
	}
	if node.GetParent() == nil {
		return false, nil
	}
	return parentMatchRecur(node.GetParent(), typeName)
}

func parentMatch(node ast.Node, typeName string) (bool, ast.Node) {
	if ok, parent := parentMatchRecur(node, typeName); ok && (node.GetChildren() == nil) {
		return ok, parent
	} else {
		return false, nil
	}
}

func postToMicroservice(serviceUrl string, text string, params map[string]string) *http.Response {
	data := url.Values{}
	data.Set("text", text)
	for k, v := range params {
		data.Set(k, v)
	}
	response, err := http.PostForm(serviceUrl, data)

	checkErr(err)

	return response
}

func getRoutes() (map[string]interface{}, error) {
	// Connection to microservices is managed via PLATFORM_ROUTES

	fmt.Println("Looking for services in PLATFORM_ROUTES")
	rawRoutes := os.Getenv("PLATFORM_ROUTES")
	jsonRoutes, _ := base64.StdEncoding.DecodeString(rawRoutes)

	var decodedRoutes map[string]interface{}

	err := json.Unmarshal([]byte(jsonRoutes), &decodedRoutes)
	if err != nil {
		return nil, err
	}

	return decodedRoutes, nil
}

func getMicroservice(serviceUrl string) (Microservice, error) {
	baseUrl, err := url.Parse(serviceUrl)
	checkErr(err)
	route, err := url.Parse("/discover")
	checkErr(err)
	referenceUrl := baseUrl.ResolveReference(route)
	response, err := http.Get(referenceUrl.String())
	checkErr(err)
	data, err := ioutil.ReadAll(response.Body)
	checkErr(err)

	var microservice Microservice
	fmt.Println(data)
	err = json.Unmarshal(data, &microservice)
	if err != nil {
		return microservice, err
	}

	fmt.Println(microservice)

	microservice.Route = serviceUrl

	fmt.Println("microservice object was created successfully")
	return microservice, nil
}

func discoverServices() ([]Microservice, error) {
	routes, err := getRoutes()
	if err != nil {
		return nil, err
	}

	var microservices []Microservice
	fmt.Println("Discovering microservices")
	for route, _ := range routes {
		if !strings.HasPrefix(route, "https://controller") && strings.HasPrefix(route, "https://") {
			microservice, err := getMicroservice(route)
			if err != nil {
				fmt.Println(err)
				continue
			}
			microservices = append(microservices, microservice)
			fmt.Println("Found microservice on cluster: ")
			fmt.Println(microservice)
		}
	}

	fmt.Println("We exited the loop")

	return microservices, nil
}

func contentToString(d1 []byte, d2 []byte) string {
	if d1 != nil {
		return string(d1)
	}
	if d2 != nil {
		return string(d2)
	}
	return ""
}

func getContent(node ast.Node) string {
	if c := node.AsContainer(); c != nil {
		return contentToString(c.Literal, c.Content)
	}
	leaf := node.AsLeaf()
	return contentToString(leaf.Literal, leaf.Content)
}
