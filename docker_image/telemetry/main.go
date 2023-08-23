package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"
	"github.com/microsoft/ApplicationInsights-Go/appinsights"
)

type telemetryClient interface {
	Track(telemetry appinsights.Telemetry)
}

func main() {
	diagnostics()
	newApp(newTelemetryClient()).Listen(fmt.Sprintf(":%d", port()))
}

func diagnostics() {
	if diagEnabledEnv := os.Getenv("DIAG"); diagEnabledEnv != "" {
		appinsights.NewDiagnosticsMessageListener(func(msg string) error {
			fmt.Printf("[%s] %s\n", time.Now().Format(time.DateTime), msg)
			return nil
		})
	}
}

func newTelemetryClient() appinsights.TelemetryClient {
	c := appinsights.NewTelemetryConfiguration(os.Getenv("INSTRUMENTATION_KEY"))
	c.MaxBatchSize = 100
	c.MaxBatchInterval = time.Second
	return appinsights.NewTelemetryClientFromConfig(c)
}

func port() int {
	port := 8080
	portEnv := os.Getenv("PORT")
	if p, err := strconv.Atoi(portEnv); portEnv != "" && err != nil {
		port = p
	}
	return port
}

func newApp(client telemetryClient) *iris.Application {
	app := iris.New()
	app.Use(iris.Compression)
	app.Post("/telemetry", func(c *context.Context) {
		var tags map[string]string
		err := c.ReadBody(&tags)
		if err != nil {
			c.StatusCode(iris.StatusBadRequest)
			c.WriteString("incorrect tags")
			return
		}
		var event, resourceId string
		var ok bool
		if event, ok = tags["event"]; !ok {
			c.StatusCode(iris.StatusBadRequest)
			c.WriteString("event required")
			return
		}
		if resourceId, ok = tags["resource_id"]; !ok {
			c.StatusCode(iris.StatusBadRequest)
			c.WriteString("resource_id required")
		}
		telemetry := appinsights.NewEventTelemetry(event)
		telemetry.Properties = tags
		telemetry.Tags.User().SetAccountId(resourceId)
		client.Track(telemetry)
		c.StatusCode(iris.StatusOK)
	})
	return app
}
