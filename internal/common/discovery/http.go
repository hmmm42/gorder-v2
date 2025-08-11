package discovery

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/hmmm42/gorder-v2/common/discovery/consul"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func RegisterHTTPToConsul(ctx context.Context, serviceName string) (func() error, error) {
	registry, err := consul.New(viper.GetString("consul.addr"))
	if err != nil {
		return func() error { return nil }, err
	}

	httpServiceName := serviceName + "-http"
	instanceID := GenerateInstanceID(httpServiceName)
	httpAddr := viper.Sub(serviceName).GetString("http-addr")

	if err := registry.Register(ctx, instanceID, httpServiceName, httpAddr, nil); err != nil {
		return func() error { return nil }, err
	}
	go func() {
		for {
			if err := registry.HealthCheck(instanceID, httpServiceName); err != nil {
				logrus.Panicf("no heartbeat from %s to registry, err=%v", httpServiceName, err)
			}
			time.Sleep(1 * time.Second)
		}
	}()
	logrus.WithFields(logrus.Fields{
		"serviceName": httpServiceName,
		"addr":        httpAddr,
	}).Info("register to consul")
	return func() error {
		return registry.Deregister(ctx, instanceID, httpServiceName)
	}, nil
}

func GetServiceAddr(ctx context.Context, serviceName string) (string, error) {
	registry, err := consul.New(viper.GetString("consul.addr"))
	if err != nil {
		return "", err
	}
	addrs, err := registry.Discover(ctx, serviceName)
	if err != nil {
		return "", err
	}
	if len(addrs) == 0 {
		return "", fmt.Errorf("got empty %s addrs from consul", serviceName)
	}
	i := rand.Intn(len(addrs))
	logrus.Infof("Discovered %d instance of %s, addrs=%v", len(addrs), serviceName, addrs)
	return addrs[i], nil
}
