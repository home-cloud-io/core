package host

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/steady-bytes/draft/pkg/chassis"
)

type (
	// Replacer take in a line in a file and outputs the replacement line (which could be the same if no change is needed)
	Replacer func(line string) string
)

var (
	ChunkPath            = "/etc/daemon/tmp"
	ConfigFile           = "/etc/home-cloud/config.yaml"
	MigrationsFile       = "/etc/home-cloud/migrations.yaml"
	WireguardKeyPath     = "/etc/home-cloud/wireguard-keys"
	NixosConfigFile      = "/etc/nixos/configuration.nix"
	NetworkingConfigFile = "/etc/nixos/config/networking.json"
	ServicesConfigFile   = "/etc/nixos/config/services.json"
	TimeConfigFile       = "/etc/nixos/config/time.json"
	DaemonNixFile        = "/etc/nixos/home-cloud/daemon/default.nix"
	DraftManifestFile    = "/var/lib/rancher/k3s/server/manifests/draft.yaml"
	OperatorManifestFile = "/var/lib/rancher/k3s/server/manifests/operator.yaml"
	ServerManifestFile   = "/var/lib/rancher/k3s/server/manifests/server.yaml"
)

var (
	// fileMutex is a safety check to make sure we don't accidentally write to the same file from multiple threads
	// in the future this could be put into a map keyed off of filenames to allow parallel writes to different files
	fileMutex = sync.Mutex{}
)

func ConfigureFilePaths(logger chassis.Logger) {
	logger.Info("configuring file paths")
	ChunkPath = FilePath(ChunkPath)
	ConfigFile = FilePath(ConfigFile)
	MigrationsFile = FilePath(MigrationsFile)
	WireguardKeyPath = FilePath(WireguardKeyPath)
	NixosConfigFile = FilePath(NixosConfigFile)
	NetworkingConfigFile = FilePath(NetworkingConfigFile)
	ServicesConfigFile = FilePath(ServicesConfigFile)
	TimeConfigFile = FilePath(TimeConfigFile)
	DaemonNixFile = FilePath(DaemonNixFile)
	DraftManifestFile = FilePath(DraftManifestFile)
	OperatorManifestFile = FilePath(OperatorManifestFile)
	ServerManifestFile = FilePath(ServerManifestFile)
}

// LineByLineReplace will process all lines in the given file running all Replacers against each line.
//
// NOTE: the Replacers will be run in the order they appear in the slice
func LineByLineReplace(filename string, replacers []Replacer) error {
	fileMutex.Lock()
	defer fileMutex.Unlock()

	// read original file
	reader, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer reader.Close()

	// create temp file
	writer, err := os.CreateTemp("", fmt.Sprintf("%s-*.tmp", filepath.Base(filename)))
	if err != nil {
		return err
	}
	defer writer.Close()

	// execute replacers (writing into the temp file)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		for _, r := range replacers {
			line = r(line)
		}
		_, err := io.WriteString(writer, line+"\n")
		if err != nil {
			return err
		}
	}
	err = scanner.Err()
	if err != nil {
		return err
	}

	// make sure the temp file was successfully written to
	err = writer.Close()
	if err != nil {
		return err
	}

	// close original file
	err = reader.Close()
	if err != nil {
		return err
	}

	// overwrite the original file with the temp file
	err = os.Rename(writer.Name(), filename)
	if err != nil {
		return err
	}

	return nil
}

// FilePath cleans the given path and makes it a local path by prefixing a "./tmp/" if
// the draft env is "test".
func FilePath(path string) string {
	if chassis.GetConfig().Env() == "test" {
		path = filepath.Join(".", "tmp", path)
	}
	return filepath.Clean(path)
}
