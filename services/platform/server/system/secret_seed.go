package system

import (
	"context"
	"time"

	kvclient "github.com/home-cloud-io/core/services/platform/server/kv-client"

	v1 "github.com/steady-bytes/draft/api/core/registry/key_value/v1"
	"github.com/steady-bytes/draft/pkg/chassis"
	"golang.org/x/exp/rand"
)

func init() {
	rand.Seed(uint64(time.Now().UnixNano()))
}

const (
	RANDOM_BYTES_LENGTH = 16

	ErrFailedToStoreSeed = "failed to create set request"
)

var (
	letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
)

func InitSecretSeed(logger chassis.Logger) {
	ctx := context.Background()

	// check if the seed already exists
	seedValue := &v1.Value{}
	err := kvclient.Get(ctx, kvclient.SEED_KEY, seedValue)
	if err != nil {
		logger.WithError(err).Warn("failed to find seed making a new one")

		// generate a random secret seed and save it to the key-value store
		seed := randStringRunes(RANDOM_BYTES_LENGTH)

		// store the seed in the key-value store
		seedValue.Data = seed

		_, err := kvclient.Set(ctx, kvclient.SEED_KEY, seedValue)
		if err != nil {
			logger.WithError(err).Error(ErrFailedToStoreSeed)
			return
		}

		logger.Info("created new secret seed")
		return
	}

	logger.Info("secret seed already exists")
}

func randStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
