package etcd

import (
	"context"
	clientv3 "go.etcd.io/etcd/client/v3"
	"time"
)

type Client struct {
	etcd *clientv3.Client
}

func NewClient(endpoints []string) (*Client, error) {
	cfg := clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	}
	etcd, err := clientv3.New(cfg)
	if err != nil {
		return nil, err
	}
	return &Client{etcd: etcd}, nil
}

func (c *Client) Close() error { return c.etcd.Close() }

func (c *Client) Get(ctx context.Context, key string) (*clientv3.GetResponse, error) {
	return c.etcd.Get(ctx, key)
}

func (c *Client) Put(ctx context.Context, key, val string) (*clientv3.PutResponse, error) {
	return c.etcd.Put(ctx, key, val)
}

func (c *Client) Delete(ctx context.Context, key string) (*clientv3.DeleteResponse, error) {
	return c.etcd.Delete(ctx, key)
}

func (c *Client) Txn(ctx context.Context) clientv3.Txn { return c.etcd.Txn(ctx) }

func (c *Client) Watch(ctx context.Context, key string, opts ...clientv3.OpOption) clientv3.WatchChan {
	return c.etcd.Watch(ctx, key, opts...)
}
