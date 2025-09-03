package cache

import (
	"context"
	"time"

	valkey "github.com/valkey-io/valkey-go"
)

// ValkeyClient implements Cache using Valkey.
type ValkeyClient struct {
	c valkey.Client
}

func NewValkey(addr, password string) (*ValkeyClient, error) {
	opts := valkey.ClientOption{
		InitAddress: []string{addr},
	}
	if password != "" {
		opts.Username = "default"
		opts.Password = password
	}
	client, err := valkey.NewClient(opts)
	if err != nil {
		return nil, err
	}
	return &ValkeyClient{c: client}, nil
}

func (v *ValkeyClient) Get(ctx context.Context, key string) (string, bool) {
	res := v.c.Do(ctx, v.c.B().Get().Key(key).Build())
	if err := res.Error(); err != nil {
		return "", false
	}
	str, err := res.ToString()
	if err != nil {
		return "", false
	}
	return str, true
}

func (v *ValkeyClient) Set(ctx context.Context, key string, val string, ttl time.Duration) error {
	if ttl > 0 {
		res := v.c.Do(ctx, v.c.B().Set().Key(key).Value(val).ExSeconds(int64(ttl/time.Second)).Build())
		return res.Error()
	}
	res := v.c.Do(ctx, v.c.B().Set().Key(key).Value(val).Build())
	return res.Error()
}

func (v *ValkeyClient) Delete(ctx context.Context, key string) error {
	res := v.c.Do(ctx, v.c.B().Del().Key(key).Build())
	return res.Error()
}

func (v *ValkeyClient) DeletePrefix(ctx context.Context, prefix string) error {
	pattern := prefix + "*"
	res := v.c.Do(ctx, v.c.B().Keys().Pattern(pattern).Build())
	if err := res.Error(); err != nil {
		return err
	}
	keys, err := res.AsStrSlice()
	if err != nil {
		return err
	}
	var lastErr error
	for _, k := range keys {
		if err := v.Delete(ctx, k); err != nil {
			lastErr = err
		}
	}
	return lastErr
}
