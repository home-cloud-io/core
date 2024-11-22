package host

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
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
	// fileMutex is a safety check to make sure we don't accidentally write to the same file from multiple threads
	// in the future this could be put into a map keyed off of filenames to allow parallel writes to different files
	fileMutex = sync.Mutex{}
)

const (
	NixosRoot        = "/etc/nixos/"
	HomeCloudRoot    = "/etc/home-cloud/"
	K3sRoot          = "/var/lib/rancher/k3s/"
	nixosConfigsPath = "config/"

	DefaultFileMode = 0600
)

// Home Cloud paths

func ChunkPath() string {
	return FilePath(HomeCloudRoot, "tmp/")
}

func ConfigFile() string {
	return FilePath(HomeCloudRoot, "config.yaml")
}

func MigrationsFile() string {
	return FilePath(HomeCloudRoot, "migrations.yaml")
}

func WireguardKeyPath() string {
	return FilePath(HomeCloudRoot, "wireguard-keys/")
}

// NixOS paths

func NixosConfigFile() string {
	return FilePath(NixosRoot, "configuration.nix")
}

func NixosVarsFile() string {
	return FilePath(NixosRoot, "vars.nix")
}

func DaemonNixFile() string {
	return FilePath(NixosRoot, "daemon/default.nix")
}

func NixosConfigsPath() string {
	return FilePath(NixosRoot, nixosConfigsPath)
}

func BootConfigFile() string {
	return FilePath(NixosRoot, nixosConfigsPath, "boot.json")
}

func NetworkingConfigFile() string {
	return FilePath(NixosRoot, nixosConfigsPath, "networking.json")
}

func SecurityConfigFile() string {
	return FilePath(NixosRoot, nixosConfigsPath, "security.json")
}

func ServicesConfigFile() string {
	return FilePath(NixosRoot, nixosConfigsPath, "services.json")
}

func TimeConfigFile() string {
	return FilePath(NixosRoot, nixosConfigsPath, "time.json")
}

func UsersConfigFile() string {
	return FilePath(NixosRoot, nixosConfigsPath, "users.json")
}

// k3s paths

func DraftManifestFile() string {
	return FilePath(K3sRoot, "server/manifests/draft.yaml")
}

func OperatorManifestFile() string {
	return FilePath(K3sRoot, "server/manifests/operator.yaml")
}

func ServerManifestFile() string {
	return FilePath(K3sRoot, "server/manifests/operator.yaml")
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
func FilePath(paths ...string) string {
	path := filepath.Join(paths...)
	if chassis.GetConfig().Env() == "test" {
		path = filepath.Join(".", "tmp", path)
	}
	return filepath.Clean(path)
}

func WriteJsonFile(path string, config any, perm fs.FileMode) error {

	bytes, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	err = os.WriteFile(path, bytes, perm)
	if err != nil {
		return err
	}

	return nil
}
