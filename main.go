package main

import (
	"context"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/goccy/go-json"
)

type ImposterModel struct {
	Imposters []struct {
		Port           int    `json:"port"`
		Protocol       string `json:"protocol"`
		RecordRequests bool   `json:"recordRequests"`
		Stubs          []struct {
			Predicates []struct {
				DeepEquals struct {
					Body    string            `json:"body"`
					Headers map[string]string `json:"headers"`
					Method  string            `json:"method"`
					Path    string            `json:"path"`
					Query   map[string]string `json:"query"`
				} `json:"deepEquals"`
				Matches struct {
					Body string `json:"body"`
					Path string `json:"path"`
				} `json:"matches"`
			} `json:"predicates"`
			Responses []struct {
				Behaviors []struct {
					Wait int `json:"wait"`
				} `json:"behaviors"`
				Is struct {
					Mode string `json:"_mode"`
					// TODO: investigate whether it can be obtained by []byte
					Body       map[string]interface{} `json:"body"`
					Headers    map[string]string      `json:"headers"`
					StatusCode int                    `json:"statusCode"`
				} `json:"is"`
			} `json:"responses"`
		} `json:"stubs"`
	} `json:"imposters"`
}

func main() {
	ctx := context.Background()
	b, err := os.ReadFile("_testdata/imposter.json")
	if err != nil {
		panic(err)
	}

	// TODO: add validation
	var imposters ImposterModel
	if err := json.UnmarshalContext(ctx, b, &imposters); err != nil {
		panic(err)
	}

	r := chi.NewRouter()
	for _, stub := range imposters.Imposters[0].Stubs {
		path := stub.Predicates[0].DeepEquals.Path
		if path == "" {
			path = stub.Predicates[0].Matches.Path
		}

		r.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			for k, v := range stub.Responses[0].Is.Headers {
				w.Header().Add(k, v)
			}

			code := stub.Responses[0].Is.StatusCode
			if code == 0 {
				code = http.StatusOK
			}
			w.WriteHeader(code)

			b, err := json.Marshal(stub.Responses[0].Is.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Write(b)
		})
	}

	http.ListenAndServe(":3000", r)
}
