package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	fpmStatus = `pool:                 api
process manager:      static
start time:           28/Dec/2016:18:06:46 +0100
start since:          65086
accepted conn:        1049662
listen queue:         0
max listen queue:     0
listen queue len:     0
idle processes:       25
active processes:     5
total processes:      30
max active processes: 30
max children reached: 0
slow requests:        0
`
	metricCount = 10
)

func TestFpmStatus(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(fpmStatus))
	})
	server := httptest.NewServer(handler)

	e := NewExporter(server.URL)
	ch := make(chan prometheus.Metric)

	go func() {
		defer close(ch)
		e.Collect(ch)
	}()

	for i := 1; i <= metricCount; i++ {
		m := <-ch
		if m == nil {
			t.Error("expected metric but got nil")
		}

	}
	if <-ch != nil {
		t.Error("expected closed channel")
	}
}
