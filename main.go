package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"time"

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
					Wait     int    `json:"wait"`
					Decorate string `json:"decorate"`
				} `json:"behaviors"`
				Is struct {
					Mode string `json:"_mode"`
					// TODO: investigate whether it can be obtained by []byte
					Body       interface{}       `json:"body"`
					Headers    map[string]string `json:"headers"`
					StatusCode int               `json:"statusCode"`
				} `json:"is"`
			} `json:"responses"`
		} `json:"stubs"`
	} `json:"imposters"`
}

const (
	gracefulShutdownTime = 10 * time.Second
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
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

	svr := &http.Server{
		Addr:    "localhost:3000",
		Handler: r,
	}
	go func() {
		if err := svr.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()

	<-ctx.Done()
	tctx, cancel := context.WithTimeout(context.Background(), gracefulShutdownTime)
	defer cancel()
	if err := svr.Shutdown(tctx); err != nil {
		panic(err)
	}
}
