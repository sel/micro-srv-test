package main

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sel/micro-srv-test/greet"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"gopkg.in/urfave/cli.v2"
)

var (
	opsProcessed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "microsrvtest_processed_ops_total",
		Help: "The total number of processed events",
	})
)

type service struct {
	log *logrus.Entry
	db  *sql.DB
}

func (s *service) Hello(ctx context.Context, req *greet.HelloRequest) (*greet.HelloResponse, error) {
	log := s.log.WithFields(logrus.Fields{"op": "Hello", "name": req.Name})
	log.Info("Called Hello")

	if err := s.queryForUser(req.Name); err != nil {
		log.Error(err)
		return nil, err
	}
	opsProcessed.Inc()
	return &greet.HelloResponse{Greeting: fmt.Sprintf("Hello %s!", req.Name)}, nil
}

func (s *service) queryForUser(userName string) error {
	//TODO: Use context for timeout
	row := s.db.QueryRow("SELECT name FROM users WHERE name=$1", userName)
	var name string
	err := row.Scan(&name)
	if err == sql.ErrNoRows {
		return fmt.Errorf("Unknown user %q", userName)
	} else if err != nil {
		return fmt.Errorf("Failed to fetch results: %v", err)
	}
	return nil
}

func main() {
	app := &cli.App{
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "grpc-port",
				Value:   ":8080",
				Usage:   "Port on which to listen for gRPC connections",
				EnvVars: []string{"GRPC_ADDRESS"},
			},
			&cli.StringFlag{
				Name:    "metrics-port",
				Value:   ":2112",
				Usage:   "Port on which to provide metrics to prometheus",
				EnvVars: []string{"METRICS_ADDRESS"},
			},
			&cli.StringFlag{
				Name:    "db-url",
				Value:   "postgres://postgres@localhost/greeting?sslmode=disable",
				Usage:   "Database connection URL",
				EnvVars: []string{"DB_URL"},
			},
		},
		Action: func(c *cli.Context) error {
			log := logrus.WithFields(logrus.Fields{})
			log.Info("Starting service")

			lis, err := net.Listen("tcp", c.String("grpc-address"))
			if err != nil {
				log.Fatalf("Failed to listen: %v", err)
			}
			grpcServer := grpc.NewServer(
				grpc.UnaryInterceptor(grpc_prometheus.UnaryServerInterceptor),
			)

			db, err := sql.Open("postgres", c.String("db-url"))
			if err != nil {
				log.Fatalf("Failed to connect to DB server")
			}
			db.SetConnMaxLifetime(0)
			db.SetMaxIdleConns(50)
			db.SetMaxOpenConns(50)
			//TODO: Use context for timeout
			if err = db.Ping(); err != nil {
				log.Fatalf("Failed to ping database: %v", err)
			}
			//TODO: Apply migrations (incl. creating the initial schema with https://github.com/golang-migrate/migrate)

			greet.RegisterGreetServer(grpcServer, &service{
				log: log,
				db:  db,
			})
			reflection.Register(grpcServer)

			go func() {
				http.Handle("/metrics", promhttp.Handler())
				log.Fatal(http.ListenAndServe(c.String("metrics-address"), nil))
			}()
			log.Infof("Service running on %s", c.String("grpc-address"))
			log.Fatal(grpcServer.Serve(lis))

			return nil
		},
	}

	app.Run(os.Args)
}
