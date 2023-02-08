package main

import (
   "net/http"
   "fmt"
   "io/ioutil"
   "encoding/json"

   "github.com/prometheus/client_golang/prometheus"
   "github.com/prometheus/client_golang/prometheus/promhttp"
)

var reg = prometheus.NewRegistry()
var m = NewMetrics(reg)
var promHandler = promhttp.HandlerFor(reg, promhttp.HandlerOpts{})

type Model []struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
	State   string `json:"state,omitempty"`
}

type metrics struct {
    models prometheus.Gauge
    state    *prometheus.GaugeVec
}


func NewMetrics(reg prometheus.Registerer) *metrics {
    m := &metrics{
        models: prometheus.NewGauge(prometheus.GaugeOpts{
            Name:      "nv_triton_models_counter",
            Help:      "Number of models in triton.",
        }),
        state: prometheus.NewGaugeVec(prometheus.GaugeOpts{
            Name:      "nv_triton_models_state",
            Help:      "Information about models.",
        },
            []string{"model_name"}),
    }
    reg.MustRegister(m.models, m.state)
    return m
}


func server() {
	url := "http://video-triton.vp-reception-dev.svc.cluster.local:8000/v2/repository/index"
    method := "POST"

    client := &http.Client {
    }
    req, err := http.NewRequest(method, url, nil)

    if err != nil {
    fmt.Println(err)
    return
    }
    res, err := client.Do(req)
    if err != nil {
    fmt.Println(err)
    return
    }
    defer res.Body.Close()

    body, err := ioutil.ReadAll(res.Body)
    if err != nil {
    fmt.Println(err)
    return
    }

    var result Model
    if err := json.Unmarshal(body, &result); err != nil {
        fmt.Println("Can not unmarshal JSON")
    }

    m.models.Set(float64(len(result)))

    for _, rec := range result {
        if rec.State == "READY" {
            m.state.With(prometheus.Labels{"model_name": rec.Name}).Set(1)
        } else {
            m.state.With(prometheus.Labels{"model_name": rec.Name}).Set(0)
        }
    }
}


func middle(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        server()
        next.ServeHTTP(w, r)
    })
}

func main() {
   finalHandler := http.Handler(promHandler)
   http.Handle("/metrics", middle(finalHandler))
   http.ListenAndServe(":8090", nil)
}