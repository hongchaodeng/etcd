package etcdbench

import (
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/integration"
)

func TestFun(t *testing.T) {
	clus := integration.NewClusterV3(t, &integration.ClusterConfig{Size: 1})
	defer clus.Terminate(t)
	etcdcli := clus.RandClient()
	motherCtx := context.TODO()

	for i := 0; i < 20; i++ {
		ctx, cancel := context.WithCancel(motherCtx)
		var wch clientv3.WatchChan
		switch i % 3 {
		case 0:
			wch = etcdcli.Watch(ctx, "/test/foo", clientv3.WithRev(1))
		case 1:
			wch = etcdcli.Watch(ctx, "/test/", clientv3.WithPrefix(), clientv3.WithRev(1))
		case 2:
			wch = etcdcli.Watch(ctx, "/test/bar", clientv3.WithRev(1))
		}
		incoming := make(chan string, 20)
		go func() {
			for resp := range wch {
				if resp.Err() != nil {
					t.Fatal(resp.Err())
				}
				for _, e := range resp.Events {
					select {
					case incoming <- string(e.Kv.Key):
					case <-ctx.Done():
						return
					}

				}
			}
			close(incoming)
		}()
		_, err := etcdcli.Put(motherCtx, "/test/foo", "1k bytes")
		if err != nil {
			t.Fatal(err)
		}

		switch i % 3 {
		case 0, 1:
			select {
			case _, ok := <-incoming:
				if !ok {
					t.Error("not ok")
				}
			case <-time.After(1 * time.Second):
				t.Error("timeout")
			}
		case 2:
			select {
			case <-incoming:
				t.Error("shoudln't from incoming chan")
			case <-time.After(300 * time.Millisecond):
			}
		}
		cancel()
	}
}
