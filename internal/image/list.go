package image

import (
	"fmt"
	"io"
	"sort"
	"text/tabwriter"
)

type Summary struct {
	Name string
	ID   string
	Size int64
}

func Images(stdout io.Writer) error {
	summaries, err := ListSummaries()
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "IMAGE\tID\tSIZE")
	for _, summary := range summaries {
		fmt.Fprintf(
			w,
			"%s\t%s\t%s\n",
			summary.Name,
			summary.ID,
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
				Name: repoTag,
				ID:   meta.ID,
				Size: meta.Size,
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
