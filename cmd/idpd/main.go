package main

import (
	"context"
	"fmt"
	nethttp "net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	influxlogger "github.com/influxdata/influxdb/logger"
	"github.com/influxdata/platform"
	"github.com/influxdata/platform/bolt"
	"github.com/influxdata/platform/chronograf/server"
	"github.com/influxdata/platform/http"
	"github.com/influxdata/platform/kit/prom"
	"github.com/influxdata/platform/nats"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func main() {
	Execute()
}

var (
	httpBindAddress   string
	authorizationPath string
	boltPath          string
)

func init() {
	viper.SetEnvPrefix("INFLUX")

	platformCmd.Flags().StringVar(&httpBindAddress, "http-bind-address", ":9999", "bind address for the rest http api")
	viper.BindEnv("HTTP_BIND_ADDRESS")
	if h := viper.GetString("HTTP_BIND_ADDRESS"); h != "" {
		httpBindAddress = h
	}

	platformCmd.Flags().StringVar(&authorizationPath, "authorizationPath", "", "path to a bootstrap token")
	viper.BindEnv("TOKEN_PATH")
	if h := viper.GetString("TOKEN_PATH"); h != "" {
		authorizationPath = h
	}

	platformCmd.Flags().StringVar(&boltPath, "bolt-path", "idpdb.bolt", "path to boltdb database")
	viper.BindEnv("BOLT_PATH")
	if h := viper.GetString("BOLT_PATH"); h != "" {
		boltPath = h
	}
}

var platformCmd = &cobra.Command{
	Use:   "idpd",
	Short: "influxdata platform",
	Run:   platformF,
}

func platformF(cmd *cobra.Command, args []string) {
	// Create top level logger
	logger := influxlogger.New(os.Stdout)

	reg := prom.NewRegistry()
	reg.MustRegister(prometheus.NewGoCollector())
	reg.WithLogger(logger)

	c := bolt.NewClient()
	c.Path = boltPath

	if err := c.Open(context.TODO()); err != nil {
		logger.Error("failed opening bolt", zap.Error(err))
		os.Exit(1)
	}
	defer c.Close()

	var authSvc platform.AuthorizationService
	{
		authSvc = c
	}

	var bucketSvc platform.BucketService
	{
		bucketSvc = c
	}

	var orgSvc platform.OrganizationService
	{
		orgSvc = c
	}

	var userSvc platform.UserService
	{
		userSvc = c
	}

	var dashboardSvc platform.DashboardService
	{
		dashboardSvc = c
	}

	chronografSvc, err := server.NewServiceV2(context.TODO(), c.DB())
	if err != nil {
		logger.Error("failed creating chronograf service", zap.Error(err))
		os.Exit(1)
	}

	errc := make(chan error)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM)

	// NATS streaming serve
	_, err = nats.CreateServer()
	if err != nil {
		logger.Error("failed to start nats streaming server", zap.Error(err))
		os.Exit(1)
	} else {
		s, err := nats.NewSubscriber("chris")
		if err != nil {
			logger.Error("failed to connect to streaming server", zap.Error(err))
			os.Exit(1)
		}
		err = s.Subscribe("subject", "group", &nats.LogHandler{Logger: logger, Name: "chris"})
		if err != nil {
			logger.Error("failed to subscribe to subject 'subject'", zap.Error(err))
			os.Exit(1)
		}

		s2, err := nats.NewSubscriber("jade")
		if err != nil {
			logger.Error("failed to connect to streaming server", zap.Error(err))
			os.Exit(1)
		}
		err = s2.Subscribe("subject", "group", &nats.LogHandler{Logger: logger, Name: "jade"})
		if err != nil {
			logger.Error("failed to subscribe to subject 'subject'", zap.Error(err))
			os.Exit(1)
		}

		p, err := nats.NewPublisher("bar")
		if err != nil {
			logger.Error("failed to connect to streaming server", zap.Error(err))
			os.Exit(1)
		}
		go func() {
			ticker := time.NewTicker(time.Millisecond * 500)
			for t := range ticker.C {
				err := p.Publish("subject", []byte("1"+t.String()))
				if err != nil {
					logger.Error("couldn't publish stuffs", zap.Error(err))
					os.Exit(1)
				}
			}
		}()

		p2, err := nats.NewPublisher("baz")
		if err != nil {
			logger.Error("failed to connect to streaming server", zap.Error(err))
			os.Exit(1)
		}
		go func() {
			ticker := time.NewTicker(time.Millisecond * 750)
			for t := range ticker.C {
				err := p2.Publish("subject", []byte("2"+t.String()))
				if err != nil {
					logger.Error("couldn't publish stuffs", zap.Error(err))
					os.Exit(1)
				}
			}
		}()
	}

	httpServer := &nethttp.Server{
		Addr: httpBindAddress,
	}

	// HTTP server
	go func() {
		bucketHandler := http.NewBucketHandler()
		bucketHandler.BucketService = bucketSvc

		orgHandler := http.NewOrgHandler()
		orgHandler.OrganizationService = orgSvc

		userHandler := http.NewUserHandler()
		userHandler.UserService = userSvc

		dashboardHandler := http.NewDashboardHandler()
		dashboardHandler.DashboardService = dashboardSvc

		authHandler := http.NewAuthorizationHandler()
		authHandler.AuthorizationService = authSvc
		authHandler.Logger = logger.With(zap.String("handler", "auth"))

		assetHandler := http.NewAssetHandler()

		// TODO(desa): what to do about idpe.
		chronografHandler := http.NewChronografHandler(chronografSvc)

		platformHandler := &http.PlatformHandler{
			BucketHandler:        bucketHandler,
			OrgHandler:           orgHandler,
			UserHandler:          userHandler,
			AuthorizationHandler: authHandler,
			DashboardHandler:     dashboardHandler,
			AssetHandler:         assetHandler,
			ChronografHandler:    chronografHandler,
		}
		reg.MustRegister(platformHandler.PrometheusCollectors()...)

		h := http.NewHandlerFromRegistry("platform", reg)
		h.Handler = platformHandler

		httpServer.Handler = h
		logger.Info("listening", zap.String("transport", "http"), zap.String("addr", httpBindAddress))
		errc <- httpServer.ListenAndServe()
	}()

	select {
	case <-sigs:
	case err := <-errc:
		logger.Fatal("unable to start platform", zap.Error(err))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	httpServer.Shutdown(ctx)
}

// Execute executes the idped command
func Execute() {
	if err := platformCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
