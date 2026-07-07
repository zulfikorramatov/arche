package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	lru "github.com/hashicorp/golang-lru/v2/expirable"
)

const (
	defaultCacheSize = 1024
	defaultCacheTTL  = 5 * time.Minute
)

type Authenticator interface {
	Authenticate(ctx context.Context, username, password string) (uuid.UUID, error)
}

type CachingAuthenticator struct {
	next    Authenticator
	cache   *lru.LRU[string, uuid.UUID]
	hmacKey []byte
}

func NewCachingAuthenticator(next Authenticator, hmacKey []byte) *CachingAuthenticator {
	return &CachingAuthenticator{
		next:    next,
		cache:   lru.NewLRU[string, uuid.UUID](defaultCacheSize, nil, defaultCacheTTL),
		hmacKey: hmacKey,
	}
}

func (c *CachingAuthenticator) key(username, password string) string {
	mac := hmac.New(sha256.New, c.hmacKey)
	mac.Write([]byte(username))
	mac.Write([]byte{':'})
	mac.Write([]byte(password))
	return hex.EncodeToString(mac.Sum(nil))
}

func (c *CachingAuthenticator) Authenticate(ctx context.Context, username, password string) (uuid.UUID, error) {
	k := c.key(username, password)
	if id, ok := c.cache.Get(k); ok {
		return id, nil
	}
	id, err := c.next.Authenticate(ctx, username, password)
	if err != nil {
		return uuid.Nil, err
	}
	c.cache.Add(k, id)
	return id, nil
}
