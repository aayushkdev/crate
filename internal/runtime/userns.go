package runtime

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
)

type idMapRange struct {
	containerID int
	hostID      int
	size        int
}

type subIDRange struct {
	start int
	count int
}

func configureRootlessUserNS(pid int) error {
	if pid <= 0 {
		return fmt.Errorf("invalid child pid %d", pid)
	}

	ranges, err := loadRootlessMappings()
	if err != nil {
		return err
	}

	if err := writeSetgroupsDeny(pid); err != nil {
		return err
	}

	if err := runIDMapHelper("newuidmap", pid, ranges.uid); err != nil {
		return err
	}
	if err := runIDMapHelper("newgidmap", pid, ranges.gid); err != nil {
		return err
	}

	return nil
}

type rootlessMappings struct {
	uid []idMapRange
	gid []idMapRange
}

func loadRootlessMappings() (*rootlessMappings, error) {
	current, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("resolve current user: %w", err)
	}

	uid, err := strconv.Atoi(current.Uid)
	if err != nil {
		return nil, fmt.Errorf("parse current uid %q: %w", current.Uid, err)
	}
	gid, err := strconv.Atoi(current.Gid)
	if err != nil {
		return nil, fmt.Errorf("parse current gid %q: %w", current.Gid, err)
	}

	subUIDs, err := readSubIDFile("/etc/subuid", current.Username, current.Uid)
	if err != nil {
		return nil, err
	}
	subGIDs, err := readSubIDFile("/etc/subgid", current.Username, current.Uid)
	if err != nil {
		return nil, err
	}

	return &rootlessMappings{
		uid: buildIDMapRanges(uid, subUIDs),
		gid: buildIDMapRanges(gid, subGIDs),
	}, nil
}

func buildIDMapRanges(hostRootID int, subIDs []subIDRange) []idMapRange {
	ranges := make([]idMapRange, 0, len(subIDs)+1)
	ranges = append(ranges, idMapRange{
		containerID: 0,
		hostID:      hostRootID,
		size:        1,
	})

	nextContainerID := 1
	for _, subID := range subIDs {
		ranges = append(ranges, idMapRange{
			containerID: nextContainerID,
			hostID:      subID.start,
			size:        subID.count,
		})
		nextContainerID += subID.count
	}

	return ranges
}

func readSubIDFile(path string, username string, numericID string) ([]subIDRange, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	var ranges []subIDRange
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Split(line, ":")
		if len(parts) != 3 {
			continue
		}
		if parts[0] != username && parts[0] != numericID {
			continue
		}

		start, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("parse %s start %q: %w", path, parts[1], err)
		}
		count, err := strconv.Atoi(parts[2])
		if err != nil {
			return nil, fmt.Errorf("parse %s count %q: %w", path, parts[2], err)
		}
		if count <= 0 {
			continue
		}

		ranges = append(ranges, subIDRange{start: start, count: count})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	if len(ranges) == 0 {
		return nil, fmt.Errorf(
			"rootless mode requires delegated IDs in %s for %s; no ranges found",
			path,
			username,
		)
	}

	return ranges, nil
}

func writeSetgroupsDeny(pid int) error {
	path := fmt.Sprintf("/proc/%d/setgroups", pid)
	if err := os.WriteFile(path, []byte("deny"), 0644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}

	return nil
}

func runIDMapHelper(helper string, pid int, ranges []idMapRange) error {
	path, err := exec.LookPath(helper)
	if err != nil {
		return fmt.Errorf("%s is required for rootless multi-ID mapping: %w", helper, err)
	}

	args := make([]string, 0, 2+len(ranges)*3)
	args = append(args, path, strconv.Itoa(pid))
	for _, r := range ranges {
		args = append(
			args,
			strconv.Itoa(r.containerID),
			strconv.Itoa(r.hostID),
			strconv.Itoa(r.size),
		)
	}

	cmd := exec.Command(path, args[1:]...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(output))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("%s failed: %s", helper, msg)
	}

	return nil
}
