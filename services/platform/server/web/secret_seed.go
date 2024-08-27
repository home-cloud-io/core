package web

import (
	"context"
	"net/http"
	"time"

	v1 "github.com/steady-bytes/draft/api/core/registry/key_value/v1"
	kvv1Connect "github.com/steady-bytes/draft/api/core/registry/key_value/v1/v1connect"
	"github.com/steady-bytes/draft/pkg/chassis"
	"golang.org/x/exp/rand"
)

func init() {
	rand.Seed(uint64(time.Now().UnixNano()))
}

const (
	RANDOM_BYTES_LENGTH = 16
	SEED_KEY            = "secret_seed"

	ErrFailedToStoreSeed = "failed to create set request"
)

var (
	letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
)

func NewSecretSeed(logger chassis.Logger) error {
	var (
		log      = logger.WithField("setup seed", "secret")
		kvClient = kvv1Connect.NewKeyValueServiceClient(http.DefaultClient, chassis.GetConfig().Entrypoint())
	)

	// generate a random secret seed and save it to the key-value store
	seed := randStringRunes(RANDOM_BYTES_LENGTH)

	// store the seed in the key-value store
	seedValue := &v1.Value{
		Data: seed,
	}
	req, err := buildSetRequest(SEED_KEY, seedValue)
	if err != nil {
		log.WithError(err).Error(ErrFailedToStoreSeed)
		return err
	}

	_, err = kvClient.Set(context.Background(), req)
	if err != nil {
		log.WithError(err).Error(ErrFailedToStoreSeed)
	}

	return nil
}

func randStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
