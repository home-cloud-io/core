package system

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	dv1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
	v1 "github.com/home-cloud-io/core/api/platform/server/v1"
	"github.com/home-cloud-io/core/services/platform/server/async"
	k8sclient "github.com/home-cloud-io/core/services/platform/server/k8s-client"
	kvclient "github.com/home-cloud-io/core/services/platform/server/kv-client"

	"github.com/containers/image/v5/docker"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/storage/memory"
	kvv1 "github.com/steady-bytes/draft/api/core/registry/key_value/v1"
	"github.com/steady-bytes/draft/pkg/chassis"
	"golang.org/x/mod/semver"
	"golang.org/x/sync/errgroup"
)

type (
	Controller interface {
		Containers
		Daemon
		Device
		Locators
		OS
		Peering
	}

	controller struct {
		k8sclient        k8sclient.System
		systemUpdateLock sync.Mutex
		broadcaster      async.Broadcaster
	}
	fileChunk struct {
		index int
		data  []byte
	}
)

func NewController(logger chassis.Logger, broadcaster async.Broadcaster) Controller {
	config := chassis.GetConfig()
	config.SetDefault(osAutoUpdateCronConfigKey, "0 1 * * *")
	config.SetDefault(containerAutoUpdateCronConfigKey, "0 2 * * *")
	return &controller{
		k8sclient:        k8sclient.NewClient(logger),
		systemUpdateLock: sync.Mutex{},
		broadcaster:      broadcaster,
	}
}

const (
	ErrFailedToBuildSeedGetRequest = "failed to build get request for seed"

	homeCloudCoreRepo                = "https://github.com/home-cloud-io/core"
	homeCloudCoreTrunk               = "main"
	daemonTagPath                    = "refs/tags/services/platform/daemon/"
	osAutoUpdateCronConfigKey        = "server.updates.os_auto_update_cron"
	containerAutoUpdateCronConfigKey = "server.updates.containers_auto_update_cron"

	// Currently only a single interface is supported and defaults to this value. In the future we
	// will probably want to support multiple interfaces (e.g. one for trusted mobile clients and another for federated servers)
	DefaultWireguardInterface = "wg0"
	// TODO: make this configurable
	DefaultSTUNServerAddress = "locator1.home-cloud.io:3478"
)

// helper functions

func (c *controller) saveSettings(ctx context.Context, logger chassis.Logger, cmd *dv1.SaveSettingsCommand) error {
	logger.Info("saving settings")
	listener := async.RegisterListener(ctx, c.broadcaster, &async.ListenerOptions[*dv1.SettingsSaved]{
		Callback: func(event *dv1.SettingsSaved) (bool, error) {
			if event.Error != "" {
				return true, errors.New(event.Error)
			}
			return true, nil
		},
		Timeout: 30 * time.Second,
	})
	err := com.Send(&dv1.ServerMessage{
		Message: &dv1.ServerMessage_SaveSettingsCommand{
			SaveSettingsCommand: cmd,
		},
	})
	if err != nil {
		return err
	}
	err = listener.Listen(ctx)
	if err != nil {
		return err
	}
	logger.Info("settings saved successfully")
	return nil
}

func getSaltValue(ctx context.Context) (string, error) {
	seedVal := &kvv1.Value{}
	err := kvclient.Get(ctx, kvclient.SEED_KEY, seedVal)
	if err != nil {
		return "", errors.New(ErrFailedToBuildSeedGetRequest)
	}

	return seedVal.GetData(), nil
}

func hashPassword(password string, salt []byte) string {
	// a little salt & hash before saving the password
	var (
		pwBytes        = []byte(password)
		sha512Hasher   = sha512.New()
		hashedPassword = sha512Hasher.Sum(nil)
	)

	pwBytes = append(pwBytes, []byte(salt)...)
	sha512Hasher.Write(pwBytes)

	return hex.EncodeToString(hashedPassword)
}

func getLatestDaemonVersion() (*v1.DaemonVersion, error) {
	var (
		latest = &v1.DaemonVersion{}
	)

	// clone repo
	repo, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
		URL:           homeCloudCoreRepo,
		ReferenceName: homeCloudCoreTrunk,
		SingleBranch:  true,
		Depth:         1,
		Tags:          git.AllTags,
	})
	if err != nil {
		return nil, err
	}

	// pull out daemon versions from tags
	iter, err := repo.Tags()
	if err != nil {
		return nil, err
	}
	versions := []string{}
	err = iter.ForEach(func(tag *plumbing.Reference) error {
		name := tag.Name().String()
		if strings.HasPrefix(name, daemonTagPath) {
			versions = append(versions, strings.TrimPrefix(name, daemonTagPath))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if len(versions) == 0 {
		return nil, fmt.Errorf("no versions found")
	}

	// sort versions by semver
	semver.Sort(versions)
	latest.Version = versions[len(versions)-1]

	// find hashes from tags
	iter, err = repo.Tags()
	if err != nil {
		return nil, err
	}
	err = iter.ForEach(func(tag *plumbing.Reference) error {
		name := tag.Name().String()
		prefix := fmt.Sprintf("refs/tags/daemon_%s", latest.Version)

		// ignore tag if it doesn't match the hash tag format for the latest version
		if !strings.HasPrefix(name, prefix) {
			return nil
		}

		// check which type of tag it is and save it
		parts := strings.Split(name, "_")
		t := parts[2]
		hash := parts[3]
		switch t {
		case "src":
			latest.SrcHash = hash
		case "vendor":
			latest.VendorHash = hash
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// grab latest
	return latest, nil
}

func getLatestImageTags(ctx context.Context, images []*v1.ImageVersion) ([]*v1.ImageVersion, error) {
	for _, image := range images {
		latest, err := getLatestImageTag(ctx, image.Image)
		if err != nil {
			return nil, err
		}
		image.Latest = latest
	}
	return images, nil
}

func getLatestImageTag(ctx context.Context, image string) (string, error) {
	ref, err := docker.ParseReference(fmt.Sprintf("//%s", image))
	if err != nil {
		return "", err
	}

	tags, err := docker.GetRepositoryTags(ctx, nil, ref)
	if err != nil {
		return "", err
	}

	semverTags := []string{}
	for _, t := range tags {
		if !semver.IsValid(t) {
			continue
		}
		semverTags = append(semverTags, t)
	}

	var latestVersion string
	if len(semverTags) > 0 {
		semver.Sort(semverTags)
		latestVersion = semverTags[len(semverTags)-1]
	}

	return latestVersion, nil
}

func (c *controller) streamFile(ctx context.Context, logger chassis.Logger, buf io.Reader, fileId string, fileName string) error {
	var (
		g         errgroup.Group
		log       = logger.WithField("file_id", fileId)
		chunkSize = 10 << 20
	)

	// create channels to communicate between goroutines
	chunks := make(chan fileChunk, 1)

	// start multiple goroutines to process chunks in parallel
	for i := 0; i < 4; i++ {
		log := log.WithField("worker", i)
		g.Go(func() error {
			log.Debug("waiting for work")
			options := &async.ListenerOptions[*dv1.UploadFileChunkCompleted]{
				Callback: func(event *dv1.UploadFileChunkCompleted) (bool, error) {
					return false, nil
				},
				Timeout: 30 * time.Minute,
				Buffer:  64,
			}
			ctxCancel, cancel := context.WithCancel(ctx)
			listener := async.RegisterListener(ctxCancel, c.broadcaster, options)
			// begin listening and close when this routine is done
			go listener.Listen(ctxCancel)
			defer cancel()

			done := make(chan bool)
			for chunk := range chunks {
				log := log.WithField("chunk_index", chunk.index)
				log.Debug("uploading chunk")

				// update callback function for the current chunk
				options.Callback = func(event *dv1.UploadFileChunkCompleted) (bool, error) {
					if event.FileId == fileId && event.Index == uint32(chunk.index) {
						done <- true
					}
					return false, nil
				}

				// upload chunk
				err := com.Send(&dv1.ServerMessage{
					Message: &dv1.ServerMessage_UploadFileRequest{
						UploadFileRequest: &dv1.UploadFileRequest{
							Data: &dv1.UploadFileRequest_Chunk{
								Chunk: &dv1.FileChunk{
									FileId: fileId,
									Index:  uint32(chunk.index),
									Data:   chunk.data,
								},
							},
						},
					},
				})
				if err != nil {
					log.WithError(err).Error("failed to send chunk to daemon")
					return err
				}

				// wait for done signal before uploading next chunk
				log.Debug("wait for done signal")
				<-done
				log.Debug("chunk upload complete")
			}
			log.Debug("done with work")
			return nil
		})
	}

	// send chunks to workers
	currentChunk := 0
	for {
		chunk := make([]byte, chunkSize)
		count, err := io.ReadFull(buf, chunk)
		if err != nil {
			// EOF means we're done
			if err == io.EOF {
				break
			}
			// error out on non-EOF error
			if err != io.ErrUnexpectedEOF {
				return err
			}
			// ErrUnexpectedEOF means we hit the end of the file before reaching chunkSize
			// so we need to trim excess bytes
			chunk = chunk[:count]
		}
		// send chunk to workers
		chunks <- fileChunk{
			index: currentChunk,
			data:  chunk,
		}
		// exit if we hit ErrUnexpectedEOF and trimmed the chunk
		if len(chunk) < chunkSize {
			break
		}
		currentChunk++
	}
	close(chunks)
	currentChunk++

	// wait for all goroutines to finish
	log.Info("waiting on pending chunks")
	err := g.Wait()
	if err != nil {
		return err
	}
	log.Info("finished uploading chunks")

	// send done signal to daemon
	err = com.Send(&dv1.ServerMessage{
		Message: &dv1.ServerMessage_UploadFileRequest{
			UploadFileRequest: &dv1.UploadFileRequest{
				Data: &dv1.UploadFileRequest_Done{
					Done: &dv1.FileDone{
						FileId:     fileId,
						ChunkCount: uint32(currentChunk),
						FilePath:   fileName,
					},
				},
			},
		},
	})
	if err != nil {
		return err
	}

	return nil
}
