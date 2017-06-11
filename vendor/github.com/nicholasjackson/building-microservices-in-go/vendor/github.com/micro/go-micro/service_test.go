package micro

import (
	"sync"
	"testing"

	"github.com/micro/go-micro/registry/mock"
	proto "github.com/micro/go-micro/server/debug/proto"

	"golang.org/x/net/context"
)

func TestService(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	// cancellation context
	ctx, cancel := context.WithCancel(context.Background())

	// create service
	service := NewService(
		Name("test.service"),
		Context(ctx),
		Registry(mock.NewRegistry()),
		AfterStart(func() error {
			wg.Done()
			return nil
		}),
	)

	// we can't test service.Init as it parses the command line
	// service.Init()

	// register handler
	// do that later

	go func() {
		// wait for start
		wg.Wait()

		// test call debug
		req := service.Client().NewRequest(
			"test.service",
			"Debug.Health",
			new(proto.HealthRequest),
		)

		rsp := new(proto.HealthResponse)

		err := service.Client().Call(context.TODO(), req, rsp)
		if err != nil {
			t.Fatal(err)
		}

		if rsp.Status != "ok" {
			t.Fatalf("service response: %s", rsp.Status)
		}

		// shutdown the service
		cancel()
	}()

	// run service
	service.Run()
}
