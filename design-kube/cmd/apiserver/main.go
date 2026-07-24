package main

import (
	"context"
	"fmt"
	"log"

	"github.com/tazzledazzle/go-cook/design-kube/pkg/storage/etcd"
)

func main() {
	client, err := etcd.NewClient([]string{"localhost:2379"})

	if err != nil {
		log.Fatalf("failed to connect to etcd: %v", err)
	}

	defer client.Close()

	ctx := context.Background()
	_, err = client.Put(ctx, "/test", "hello")

	if err != nil {
		log.Fatalf("put failed: %v", err)
	}

	resp, err := client.Get(ctx, "/test")

	if err != nil {
		log.Fatalf("get failed: %v", err)
	}

	fmt.Printf("etcd connectivity OK - got: %s/\n", resp.Kvs[0].Value)
}
