// The http_server command was automatically generated by Shenzhen Go.
package main

import (
	"context"
	"fmt"
	"github.com/google/shenzhen-go/dev/parts"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"html/template"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"math/cmplx"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"sync"
	"time"
)

var _ = runtime.Compiler

var (
	httpServeMuxRequestsIn = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "shenzhen_go",
			Subsystem: "httpservemux",
			Name:      "requests_in",
			Help:      "Requests received by HTTPServeMux nodes.",
		},
		[]string{"node_name", "instance_num"},
	)
	httpServeMuxRequestsOut = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "shenzhen_go",
			Subsystem: "httpservemux",
			Name:      "requests_out",
			Help:      "Requests sent out of HTTPServeMux nodes.",
		},
		[]string{"node_name", "instance_num", "output_pin"},
	)
)

func init() {
	prometheus.MustRegister(
		httpServeMuxRequestsIn,
		httpServeMuxRequestsOut,
	)
}

func Duration(in <-chan *parts.HTTPRequest, out chan<- *parts.HTTPRequest) {
	// Duration
	multiplicity := runtime.NumCPU()

	sum := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "shenzhen_go",
			Subsystem: "instrument_handler",
			Name:      "Duration",
			Help:      "Durations of requests",
			Buckets:   []float64(nil),
		},
		[]string(nil))
	prometheus.MustRegister(sum)

	defer func() {
		close(out)
	}()
	var multWG sync.WaitGroup
	multWG.Add(multiplicity)
	defer multWG.Wait()
	for n := 0; n < multiplicity; n++ {
		go func() {
			defer multWG.Done()

			h := promhttp.InstrumentHandlerDuration(sum, parts.HTTPHandler(out))
			for r := range in {
				h.ServeHTTP(r.ResponseWriter, r.Request)
				r.Close()
			}
		}()
	}
}

/* Generates a Mandelbrot fractal */
func Generate_a_Mandelbrot(requests <-chan *parts.HTTPRequest) {
	// Generate a Mandelbrot
	multiplicity := runtime.NumCPU()
	const tileW = 320
	const depth = 25

	var multWG sync.WaitGroup
	multWG.Add(multiplicity)
	defer multWG.Wait()
	for n := 0; n < multiplicity; n++ {
		go func() {
			defer multWG.Done()
			for req := range requests {
				func() {
					defer req.Close()

					q := req.Request.URL.Query()
					tileX, e0 := strconv.Atoi(q.Get("x"))
					tileY, e1 := strconv.Atoi(q.Get("y"))
					zoom, e2 := strconv.ParseUint(q.Get("z"), 10, 64)
					if e0 != nil || e1 != nil || e2 != nil || zoom > 50 {
						http.Error(req, "invalid parameter", http.StatusBadRequest)
						return
					}
					zoom = 1 << zoom
					offset := complex(float64(tileX), float64(tileY))

					req.Header().Set("Content-Type", "image/png")
					req.WriteHeader(http.StatusOK)

					img := image.NewRGBA(image.Rect(0, 0, tileW, tileW))

					for i := 0; i < tileW; i++ {
						for j := 0; j < tileW; j++ {
							c := complex(float64(i), float64(j))
							c /= tileW
							c += offset
							c *= 2
							c /= complex(float64(zoom), 0)

							z := 0i

							col := color.Black
							for k := 0; k < depth; k++ {
								z = z*z + c

								// Higher escape radius makes it smoother
								if mz := cmplx.Abs(z); mz > 50 {
									sm := float64(k) + 1 - math.Log2(math.Log(mz))
									col = color.Gray16{uint16(sm * 65536 / depth)}
									break
								}
							}
							img.Set(i, j, col)
						}
					}

					png.Encode(req, img)
				}()
			}
		}()
	}
}

func HTTP_Server(errors chan<- error, manager <-chan parts.HTTPServerManager, requests chan<- *parts.HTTPRequest) {
	// HTTP Server

	defer func() {
		close(requests)
		if errors != nil {
			close(errors)
		}
	}()

	for mgr := range manager {
		svr := &http.Server{
			Handler: parts.HTTPHandler(requests),
			Addr:    mgr.Addr(),
		}
		done := make(chan struct{})
		go func() {
			if err := svr.ListenAndServe(); err != nil && errors != nil {
				errors <- err
			}
			close(done)
		}()
		if err := svr.Shutdown(mgr.Wait()); err != nil && errors != nil {
			errors <- err
		}
		<-done
	}
}

func Handle_(requests <-chan *parts.HTTPRequest) {
	// Handle /
	multiplicity := runtime.NumCPU()
	tmpl := template.Must(template.New("root").Parse(`<html>
<head>
	<title>Mandelbrot viewer</title>
	<style><!--
		img {
			float: left;
		}
		img.first {
			clear: left;
		}
		img:hover {
			border: thick red;
		}
	--></style>
</head>
<body>
	<img src="/mandelbrot?x={{.X}}&y={{.Y}}&z={{.Z}}" class="first" />
	<img src="/mandelbrot?x={{.X1}}&y={{.Y}}&z={{.Z}}" />
	<img src="/mandelbrot?x={{.X}}&y={{.Y1}}&z={{.Z}}" class="first" />
	<img src="/mandelbrot?x={{.X1}}&y={{.Y1}}&z={{.Z}}" />
</body>
</html>`))

	type params struct {
		X, X1, Y, Y1 int
		Z            uint
	}
	var multWG sync.WaitGroup
	multWG.Add(multiplicity)
	defer multWG.Wait()
	for n := 0; n < multiplicity; n++ {
		go func() {
			defer multWG.Done()
			for r := range requests {
				func() {
					defer r.Close()
					p := params{X: -1, X1: 0, Y: -1, Y1: 0, Z: 0}
					q := r.Request.URL.Query()
					if xs := q.Get("x"); xs != "" {
						x, err := strconv.Atoi(xs)
						if err != nil {
							http.Error(r, "invalid x parameter", http.StatusBadRequest)
							return
						}
						p.X, p.X1 = x, x+1
					}
					if ys := q.Get("y"); ys != "" {
						y, err := strconv.Atoi(ys)
						if err != nil {
							http.Error(r, "invalid y parameter", http.StatusBadRequest)
							return
						}
						p.Y, p.Y1 = y, y+1
					}
					if zs := q.Get("z"); zs != "" {
						z, err := strconv.ParseUint(q.Get("z"), 10, 64)
						if err != nil {
							http.Error(r, "invalid z parameter", http.StatusBadRequest)
							return
						}
						p.Z = uint(z)
					}
					if err := tmpl.Execute(r, p); err != nil {
						panic(err)
					}
				}()
			}
		}()
	}
}

func Log_errors(errors <-chan error) {
	// Log errors

	for err := range errors {
		log.Printf("HTTP server: %v", err)
	}
}

func Mandelbrot_duration(in <-chan *parts.HTTPRequest, out chan<- *parts.HTTPRequest) {
	// Mandelbrot duration
	multiplicity := runtime.NumCPU()

	sum := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "shenzhen_go",
			Subsystem: "instrument_handler",
			Name:      "Mandelbrot_duration",
			Help:      "Durations of requests",
			Buckets:   []float64(nil),
		},
		[]string(nil))
	prometheus.MustRegister(sum)

	defer func() {
		close(out)
	}()
	var multWG sync.WaitGroup
	multWG.Add(multiplicity)
	defer multWG.Wait()
	for n := 0; n < multiplicity; n++ {
		go func() {
			defer multWG.Done()

			h := promhttp.InstrumentHandlerDuration(sum, parts.HTTPHandler(out))
			for r := range in {
				h.ServeHTTP(r.ResponseWriter, r.Request)
				r.Close()
			}
		}()
	}
}

func Metrics(requests <-chan *parts.HTTPRequest) {
	// Metrics
	multiplicity := runtime.NumCPU()

	var multWG sync.WaitGroup
	multWG.Add(multiplicity)
	defer multWG.Wait()
	for n := 0; n < multiplicity; n++ {
		go func() {
			defer multWG.Done()
			h := promhttp.Handler()
			for r := range requests {
				h.ServeHTTP(r.ResponseWriter, r.Request)
				r.Close()
			}
		}()
	}
}

func Mux(mandelbrot chan<- *parts.HTTPRequest, metrics chan<- *parts.HTTPRequest, requests <-chan *parts.HTTPRequest, root chan<- *parts.HTTPRequest) {
	// Mux
	multiplicity := runtime.NumCPU()
	mux := http.NewServeMux()
	outLabels := make(map[parts.HTTPHandler]string)
	mux.Handle("/", parts.HTTPHandler(root))
	outLabels[root] = "root"
	mux.Handle("/mandelbrot", parts.HTTPHandler(mandelbrot))
	outLabels[mandelbrot] = "mandelbrot"
	mux.Handle("/metrics", parts.HTTPHandler(metrics))
	outLabels[metrics] = "metrics"

	defer func() {
		close(root)
		close(mandelbrot)
		close(metrics)

	}()
	var multWG sync.WaitGroup
	multWG.Add(multiplicity)
	defer multWG.Wait()
	for n := 0; n < multiplicity; n++ {
		instanceNumber := n
		go func() {
			defer multWG.Done()

			labels := prometheus.Labels{
				"node_name":    "Mux",
				"instance_num": strconv.Itoa(instanceNumber),
			}
			reqsIn := httpServeMuxRequestsIn.With(labels)
			reqsOut := httpServeMuxRequestsOut.MustCurryWith(labels)
			for req := range requests {
				reqsIn.Inc()
				// Borrow fix for Go issues #3692 and #5955.
				if req.Request.RequestURI == "*" {
					if req.Request.ProtoAtLeast(1, 1) {
						req.ResponseWriter.Header().Set("Connection", "close")
					}
					req.ResponseWriter.WriteHeader(http.StatusBadRequest)
					req.Close()
					continue
				}
				h, _ := mux.Handler(req.Request)
				hh, ok := h.(parts.HTTPHandler)
				if !ok {
					// ServeMux may return handlers that weren't added in the head.
					h.ServeHTTP(req.ResponseWriter, req.Request)
					req.Close()
					continue
				}
				reqsOut.With(prometheus.Labels{"output_pin": outLabels[hh]}).Inc()
				hh <- req
			}
		}()
	}
}

func Server_manager(manager chan<- parts.HTTPServerManager) {
	// Server manager

	defer func() {
		close(manager)
	}()
	mgr := parts.NewHTTPServerManager(":8765")
	manager <- mgr

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	fmt.Println("Press Ctrl-C (or SIGINT) to shut down.")
	<-sig

	timeout := 5 * time.Second
	fmt.Printf("Shutting down within %v...\n", timeout)
	ctx, canc := context.WithTimeout(context.Background(), timeout)
	mgr.Shutdown(ctx)
	go func() {
		time.Sleep(timeout)
		canc()
	}()
}

func main() {

	channel0 := make(chan *parts.HTTPRequest, 0)
	channel1 := make(chan parts.HTTPServerManager, 0)
	channel2 := make(chan error, 0)
	channel3 := make(chan *parts.HTTPRequest, 0)
	channel4 := make(chan *parts.HTTPRequest, 0)
	channel5 := make(chan *parts.HTTPRequest, 0)
	channel6 := make(chan *parts.HTTPRequest, 0)
	channel7 := make(chan *parts.HTTPRequest, 0)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		Duration(channel4, channel5)
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		Generate_a_Mandelbrot(channel7)
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		HTTP_Server(channel2, channel1, channel0)
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		Handle_(channel5)
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		Log_errors(channel2)
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		Mandelbrot_duration(channel6, channel7)
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		Metrics(channel3)
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		Mux(channel6, channel3, channel0, channel4)
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		Server_manager(channel1)
		wg.Done()
	}()

	// Wait for the various goroutines to finish.
	wg.Wait()
}
