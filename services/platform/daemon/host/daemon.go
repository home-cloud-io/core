package host

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	v1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
	"github.com/home-cloud-io/core/services/platform/daemon/execute"

	"github.com/steady-bytes/draft/pkg/chassis"
	"golang.org/x/mod/semver"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

const (
	versionPrefix    = "  version = \""
	vendorHashPrefix = "  vendorHash = \""
	srcHashPrefix    = "    hash = \""
	nixLineSuffix    = "\";"

	mockJournalLogs = `
2025-02-06T14:15:21.735028-06:00 home-cloud daemon[403489]: 2:15PM INF shutting down function=shutdown service=home-cloud-daemon
2025-02-06T14:15:21.735028-06:00 home-cloud daemon[403489]: 2:15PM INF shutdown successfully function=shutdown service=home-cloud-daemon
2025-02-06T14:15:21.736642-06:00 home-cloud systemd[1]: Stopping Home Cloud Daemon...
2025-02-06T14:15:21.738464-06:00 home-cloud systemd[1]: daemon.service: Deactivated successfully.
2025-02-06T14:15:21.745153-06:00 home-cloud systemd[1]: Stopped Home Cloud Daemon.
2025-02-06T14:15:21.745432-06:00 home-cloud systemd[1]: daemon.service: Consumed 178ms CPU time, 9.8M memory peak, 8K read from disk, 6.2K incoming IP traffic, 19.1K outgoing IP traffic.
2025-02-06T14:15:22.610482-06:00 home-cloud systemd[1]: Started Home Cloud Daemon.
2025-02-06T14:15:22.645668-06:00 home-cloud daemon[403678]: 2:15PM INF running server on: localhost:9000 function=runMux service=home-cloud-daemon
2025-02-06T14:15:22.645668-06:00 home-cloud daemon[403678]: 2:15PM INF starting mDNS publishing function=Start service=home-cloud-daemon
2025-02-06T14:15:22.652157-06:00 home-cloud daemon[403678]: 2:15PM INF starting function=Listen service=home-cloud-daemon
2025-02-06T14:15:22.652157-06:00 home-cloud daemon[403678]: 2:15PM INF running migrations function=Migrate service=home-cloud-daemon
2025-02-06T14:15:22.652157-06:00 home-cloud daemon[403678]: 2:15PM INF listening for messages from server function=listen service=home-cloud-daemon
2025-02-06T14:15:22.652157-06:00 home-cloud daemon[403678]: 2:15PM INF migrations completed function=Migrate service=home-cloud-daemon
2025-02-06T14:15:22.656777-06:00 home-cloud daemon[403678]: 2:15PM INF found outbound IP address address=192.168.1.183 function=Start service=home-cloud-daemon
2025-02-06T14:15:22.656777-06:00 home-cloud daemon[403678]: 2:15PM INF publishing hostname to mDNS fqdn=hello.local function=register service=home-cloud-daemon
2025-02-06T14:15:22.656777-06:00 home-cloud daemon[403678]: 2:15PM INF publishing hostname to mDNS fqdn=memos.local function=register service=home-cloud-daemon
2025-02-06T14:15:22.656777-06:00 home-cloud daemon[403678]: 2:15PM INF publishing hostname to mDNS fqdn=movies.local function=register service=home-cloud-daemon
2025-02-06T14:15:22.656777-06:00 home-cloud daemon[403678]: 2:15PM INF publishing hostname to mDNS fqdn=recipes.local function=register service=home-cloud-daemon
2025-02-06T14:15:22.656777-06:00 home-cloud daemon[403678]: 2:15PM INF publishing hostname to mDNS fqdn=search.local function=register service=home-cloud-daemon
2025-02-06T14:15:22.656777-06:00 home-cloud daemon[403678]: 2:15PM INF publishing hostname to mDNS fqdn=homeassistant.local function=register service=home-cloud-daemon
2025-02-06T14:15:22.656777-06:00 home-cloud daemon[403678]: 2:15PM INF publishing hostname to mDNS fqdn=photos.local function=register service=home-cloud-daemon
2025-02-06T14:15:23.680434-06:00 home-cloud daemon[403678]: Established under name 'photos.local'
2025-02-06T14:15:23.681921-06:00 home-cloud daemon[403678]: Established under name 'hello.local'
2025-02-06T14:15:23.683508-06:00 home-cloud daemon[403678]: Established under name 'recipes.local'
2025-02-06T14:15:23.685116-06:00 home-cloud daemon[403678]: Established under name 'memos.local'
2025-02-06T14:15:23.686497-06:00 home-cloud daemon[403678]: Established under name 'homeassistant.local'
2025-02-06T14:15:23.687791-06:00 home-cloud daemon[403678]: Established under name 'movies.local'
2025-02-06T14:15:23.689101-06:00 home-cloud daemon[403678]: Established under name 'search.local'
`
)

var (
	// source: https://semver.org/#is-there-a-suggested-regular-expression-regex-to-check-a-semver-string
	semverRegex = regexp.MustCompile(`v(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?`)
)

func GetDaemonVersion(logger chassis.Logger) (*v1.CurrentDaemonVersion, error) {
	f, err := os.Open(DaemonNixFile())
	if err != nil {
		logger.WithError(err).Error("failed to read daemon nix file")
		return nil, err
	}
	defer f.Close()

	current := &v1.CurrentDaemonVersion{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, versionPrefix) {
			current.Version = semverRegex.FindString(line)
			continue
		}
		if strings.HasPrefix(line, vendorHashPrefix) {
			current.VendorHash = strings.TrimSuffix(strings.TrimPrefix(line, vendorHashPrefix), nixLineSuffix)
			continue
		}
		if strings.HasPrefix(line, srcHashPrefix) {
			current.SrcHash = strings.TrimSuffix(strings.TrimPrefix(line, srcHashPrefix), nixLineSuffix)
			continue
		}
	}

	if !semver.IsValid(current.Version) {
		return nil, fmt.Errorf("invalid daemon version: %s", current.Version)
	}

	return current, nil
}

// TODO-RC2: There's a bit of a race condition with this right now. If you call GetOSVersionDiff and then call this
// method you'll accidentally upgrade the entire OS with any changes that were pulled in from the `nix-channel --update`
// that was run during GetOSVersionDiff. This can be avoided by running `nix-channel --rollback` but will require some
// stateful logic which checks if a rollback is really needed. It's out of scope for RC1 but should be revisited later.
func ChangeDaemonVersion(ctx context.Context, logger chassis.Logger, def *v1.ChangeDaemonVersionCommand) error {
	var (
		err       error
		replacers = []Replacer{
			func(line ReplacerLine) string {
				if strings.Contains(line.Current, "version =") {
					line.Current = fmt.Sprintf("  version = \"%s\";", def.Version)
				}
				return line.Current
			},
			func(line ReplacerLine) string {
				if strings.Contains(line.Current, "vendorHash =") {
					line.Current = fmt.Sprintf("  vendorHash = \"%s\";", def.VendorHash)
				}
				return line.Current
			},
			func(line ReplacerLine) string {
				if strings.Contains(line.Current, "hash =") {
					// NOTE: the double indentation is deliberate
					line.Current = fmt.Sprintf("    hash = \"%s\";", def.SrcHash)
				}
				return line.Current
			},
		}
	)

	err = LineByLineReplace(DaemonNixFile(), replacers)
	if err != nil {
		logger.WithError(err).Error("failed to replace version")
		return err
	}

	err = RebuildAndSwitchOS(ctx, logger)
	if err != nil {
		return err
	}

	return nil
}

func DaemonLogs(ctx context.Context, logger chassis.Logger, sinceSeconds uint32) ([]*v1.Log, error) {
	var (
		logs = []*v1.Log{}
	)

	raw, err := journalLogs(ctx, logger, "daemon", sinceSeconds)
	if err != nil {
		return logs, err
	}

	scanner := bufio.NewScanner(strings.NewReader(raw))
	for scanner.Scan() {
		s := strings.SplitN(scanner.Text(), " home-cloud ", 2)
		if len(s) != 2 {
			continue
		}
		t, err := time.Parse(time.RFC3339Nano, s[0])
		if err != nil {
			continue
		}
		logs = append(logs, &v1.Log{
			Source:    "daemon",
			Namespace: "host",
			Domain:    "platform",
			Log:       s[1],
			Timestamp: timestamppb.New(t),
		})
	}

	return logs, nil
}

func journalLogs(ctx context.Context, logger chassis.Logger, unit string, sinceSeconds uint32) (string, error) {
	var (
		cmd *exec.Cmd
	)

	config := chassis.GetConfig()
	if config.Env() == "test" {
		logger.Info("mocking journal logs")
		return mockJournalLogs, nil
	}

	logger.Debug("getting journal logs")
	cmd = execute.NewElevatedCommand("journalctl", "-u", unit, "--since", fmt.Sprintf("%dsec ago", sinceSeconds), "-o", "short-iso-precise")
	output, err := execute.ExecuteCommandReturnStdout(ctx, cmd)
	if err != nil {
		logger.WithError(err).Error("failed to get journal logs")
		return "", err
	}
	logger.Debug("successfully got journal logs")

	return output, nil
}
