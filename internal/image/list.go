package image

import (
	"fmt"
	"io"
	"sort"
	"text/tabwriter"
	"time"
)

type Summary struct {
	Name           string
	ID             string
	ManifestDigest string
	Platform       string
	Layers         int
	Created        time.Time
	Size           int64
}

func Images(stdout io.Writer) error {
	summaries, err := ListSummaries()
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "IMAGE\tID\tMANIFEST\tPLATFORM\tLAYERS\tCREATED\tSIZE")
	for _, summary := range summaries {
		fmt.Fprintf(
			w,
			"%s\t%s\t%s\t%s\t%d\t%s\t%s\n",
			summary.Name,
			summary.ID,
			shortImageID(summary.ManifestDigest),
			summary.Platform,
			summary.Layers,
			formatCreated(summary.Created),
			formatSize(summary.Size),
		)
	}

	return w.Flush()
}

func ListSummaries() ([]Summary, error) {
	metas, err := readAllMetadata()
	if err != nil {
		return nil, err
	}

	summaries := make([]Summary, 0, len(metas))
	for _, meta := range metas {
		tags := meta.RepoTags
		if len(tags) == 0 {
			tags = []string{"<none>:<none>"}
		}

		for _, repoTag := range tags {
			summaries = append(summaries, Summary{
				Name:           repoTag,
				ID:             meta.ID,
				ManifestDigest: meta.ManifestDigest,
				Platform:       formatPlatform(meta.OS, meta.Architecture),
				Layers:         len(meta.Layers),
				Created:        meta.Created,
				Size:           meta.Size,
			})
		}
	}

	sort.Slice(summaries, func(i, j int) bool {
		if summaries[i].Name != summaries[j].Name {
			return summaries[i].Name < summaries[j].Name
		}
		return summaries[i].ID < summaries[j].ID
	})

	return summaries, nil
}

func formatPlatform(osName, arch string) string {
	if osName == "" && arch == "" {
		return "-"
	}
	if osName == "" {
		return arch
	}
	if arch == "" {
		return osName
	}
	return osName + "/" + arch
}

func formatCreated(created time.Time) string {
	if created.IsZero() {
		return "-"
	}

	return created.Local().Format("2006-01-02 15:04")
}

func formatSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}

	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.2f %ciB", float64(size)/float64(div), "KMGTPE"[exp])
}
