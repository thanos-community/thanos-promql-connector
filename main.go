package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"math"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/gogo/status"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/model"
	"github.com/thanos-io/thanos/pkg/api/query/querypb"
	"github.com/thanos-io/thanos/pkg/info/infopb"
	"github.com/thanos-io/thanos/pkg/store/labelpb"
	"github.com/thanos-io/thanos/pkg/store/storepb/prompb"
	"google.golang.org/api/option"
	apihttp "google.golang.org/api/transport/http"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

var (
	credentialsFile = flag.String("query.credentials-file", "",
		"JSON-encoded credentials (service account or refresh token). Can be left empty if default credentials have sufficient permission.")
	targetURL = flag.String("query.target-url", "",
		"The URL to forward query requests to.")
	connectorAddress = flag.String("connector-address", ":8081",
		"Address on which to expose the query grpc server.")
	metricsAddress = flag.String("metrics-address", ":9090",
		"Address on which to expose metrics")
	queryBackendClient v1.API
)

type queryServer struct{}

func (*queryServer) Query(req *querypb.QueryRequest, srv querypb.Query_QueryServer) error {
	ts := time.Unix(req.TimeSeconds, 0)
	timeout := time.Duration(req.TimeoutSeconds) * time.Second
	values, warnings, err := queryBackendClient.Query(srv.Context(), req.Query, ts, v1.WithTimeout(timeout))
	if err != nil {
		return status.Error(codes.Aborted, err.Error())
	}
	if len(warnings) > 0 {
		errs := make([]error, 0, len(warnings))
		for _, warning := range warnings {
			errs = append(errs, errors.New(warning))
		}
		if err = srv.SendMsg(querypb.NewQueryWarningsResponse(errs...)); err != nil {
			return err
		}
	}
	switch results := values.(type) {
	case model.Vector:
		for _, result := range results {
			series := &prompb.TimeSeries{
				Samples: []prompb.Sample{{Value: float64(result.Value), Timestamp: int64(result.Timestamp)}},
				Labels:  zLabelsFromMetric(result.Metric),
			}
			if err := srv.Send(querypb.NewQueryResponse(series)); err != nil {
				return err
			}
		}
	case *model.Scalar:
		series := &prompb.TimeSeries{Samples: []prompb.Sample{{Value: float64(results.Value), Timestamp: int64(results.Timestamp)}}}
		return srv.Send(querypb.NewQueryResponse(series))
	}
	return nil
}

func (*queryServer) QueryRange(req *querypb.QueryRangeRequest, srv querypb.Query_QueryRangeServer) error {
	timeout := time.Duration(req.TimeoutSeconds) * time.Second
	interval := v1.Range{
		Start: time.Unix(req.StartTimeSeconds, 0),
		End:   time.Unix(req.EndTimeSeconds, 0),
		Step:  time.Duration(req.IntervalSeconds) * time.Second}
	values, warnings, err := queryBackendClient.QueryRange(srv.Context(), req.Query, interval, v1.WithTimeout(timeout))
	if err != nil {
		return status.Error(codes.Aborted, err.Error())
	}
	if len(warnings) > 0 {
		errs := make([]error, 0, len(warnings))
		for _, warning := range warnings {
			errs = append(errs, errors.New(warning))
		}
		if err = srv.SendMsg(querypb.NewQueryRangeWarningsResponse(errs...)); err != nil {
			return err
		}
	}
	switch results := values.(type) {
	case model.Matrix:
		for _, result := range results {
			series := &prompb.TimeSeries{
				Samples: samplesFromModel(result.Values),
				Labels:  zLabelsFromMetric(result.Metric),
			}
			if err := srv.Send(querypb.NewQueryRangeResponse(series)); err != nil {
				return err
			}
		}
	case model.Vector:
		for _, result := range results {
			series := &prompb.TimeSeries{
				Samples: []prompb.Sample{{Value: float64(result.Value), Timestamp: int64(result.Timestamp)}},
				Labels:  zLabelsFromMetric(result.Metric),
			}
			if err := srv.Send(querypb.NewQueryRangeResponse(series)); err != nil {
				return err
			}
		}
	case *model.Scalar:
		series := &prompb.TimeSeries{Samples: []prompb.Sample{{Value: float64(results.Value), Timestamp: int64(results.Timestamp)}}}
		return srv.Send(querypb.NewQueryRangeResponse(series))
	}
	return nil
}

// samplesFromModel converts model.SamplePair to prompb.Sample.
func samplesFromModel(samples []model.SamplePair) []prompb.Sample {
	result := make([]prompb.Sample, 0, len(samples))
	for _, s := range samples {
		result = append(result, prompb.Sample{
			Value:     float64(s.Value),
			Timestamp: int64(s.Timestamp),
		})
	}
	return result
}

// zLabelsFromMetric converts model.Metric to labelpb.Zlabel.
func zLabelsFromMetric(metric model.Metric) []labelpb.ZLabel {
	zlabels := make([]labelpb.ZLabel, 0, len(metric))
	for labelName, labelValue := range metric {
		zlabel := labelpb.ZLabel{Name: string(labelName), Value: string(labelValue)}
		zlabels = append(zlabels, zlabel)
	}
	return zlabels
}

type infoServer struct{}

func (*infoServer) Info(ctx context.Context, in *infopb.InfoRequest) (*infopb.InfoResponse, error) {
	return &infopb.InfoResponse{
		Query: &infopb.QueryAPIInfo{},
		Store: &infopb.StoreInfo{
			MinTime:                      math.MinInt64,
			MaxTime:                      math.MaxInt64,
			SupportsWithoutReplicaLabels: true,
			TsdbInfos: []infopb.TSDBInfo{
				{
					MinTime: math.MinInt64,
					MaxTime: math.MaxInt64,
				},
			},
		},
	}, nil
}

// createGCMServer creates a
func createGCMServer() (v1.API, error) {
	opts := []option.ClientOption{
		option.WithScopes("https://www.googleapis.com/auth/monitoring.read"),
		option.WithCredentialsFile(*credentialsFile),
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	transport, err := apihttp.NewTransport(ctx, http.DefaultTransport, opts...)
	if err != nil {
		return nil, fmt.Errorf("error creating proxy HTTP transport: %s", err)
	}
	client, err := api.NewClient(api.Config{
		Address:      *targetURL,
		RoundTripper: transport,
	})
	if err != nil {
		return nil, fmt.Errorf("error creating client: %s", err)
	}
	return v1.NewAPI(client), nil
}

func main() {
	flag.Parse()
	logger := log.NewJSONLogger(log.NewSyncWriter(os.Stderr))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)
	logger = log.With(logger, "caller", log.DefaultCaller)

	if *targetURL == "" {
		level.Error(logger).Log("msg", "--query.target-url must be set")
		os.Exit(1)
	}
	var err error
	queryBackendClient, err = createGCMServer()
	if err != nil {
		level.Error(logger).Log("msg", "Error creating client", "err", err)
		os.Exit(1)
	}

	var g run.Group
	{
		term := make(chan os.Signal, 1)
		cancel := make(chan struct{})
		signal.Notify(term, os.Interrupt, syscall.SIGTERM)

		g.Add(
			func() error {
				select {
				case <-term:
					level.Info(logger).Log("msg", "received SIGTERM, exiting gracefully...")
				case <-cancel:
				}
				return nil
			},
			func(err error) {
				close(cancel)
			},
		)
	}
	{
		//grpc server.
		listener, err := net.Listen("tcp", *connectorAddress)
		if err != nil {
			panic(err)
		}
		server := grpc.NewServer()
		querypb.RegisterQueryServer(server, &queryServer{})
		infopb.RegisterInfoServer(server, &infoServer{})
		g.Add(func() error {
			level.Info(logger).Log("msg", "Starting grpc server for query endpoint", "listen", *connectorAddress)
			return server.Serve(listener)
		}, func(err error) {
			server.GracefulStop()
		})
	}
	{
		// http server.
		ctx, cancel := context.WithCancel(context.Background())
		server := &http.Server{Addr: *metricsAddress}
		http.Handle("/metrics", promhttp.Handler())

		http.HandleFunc("/-/healthy", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "OK")
		})
		http.HandleFunc("/-/ready", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "OK")
		})

		g.Add(func() error {
			level.Info(logger).Log("msg", "Starting web server for metrics", "listen", *metricsAddress)
			return server.ListenAndServe()
		}, func(err error) {
			server.Shutdown(ctx)
			cancel()
		})
	}

	if err := g.Run(); err != nil {
		level.Error(logger).Log("msg", "running reloader failed", "err", err)
		os.Exit(1)
	}
}
