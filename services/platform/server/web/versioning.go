package web

import (
	"context"
	"fmt"
	"strings"

	v1 "github.com/home-cloud-io/core/api/platform/server/v1"

	"github.com/containers/image/v5/docker"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/storage/memory"
	"golang.org/x/mod/semver"
)

const (
	homeCloudCoreRepo  = "https://github.com/home-cloud-io/core"
	homeCloudCoreTrunk = "main"
)

func getLatestDaemonVersion() (string, error) {
	// clone repo
	repo, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
		URL:           homeCloudCoreRepo,
		ReferenceName: homeCloudCoreTrunk,
		SingleBranch:  true,
		Depth:         1,
		Tags:          git.AllTags,
	})
	if err != nil {
		return "", err
	}

	// pull out daemon versions from tags
	iter, err := repo.Tags()
	if err != nil {
		return "", err
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
		return "", err
	}

	if len(versions) == 0 {
		return "", fmt.Errorf("no versions found")
	}

	// sort versions by semver
	semver.Sort(versions)

	// grab latest
	return versions[len(versions)-1], nil
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
