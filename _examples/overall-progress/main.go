// Command overall-progress downloads "bv+ba" and prints one monotonic progress
// line across both streams, the merge and the final move.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/saeidrp/go-ytdlp"
)

func main() {
	ytdlp.MustInstallAll(context.TODO())

	url := "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
	if len(os.Args) > 1 {
		url = os.Args[1]
	}

	_, err := ytdlp.New().
		Format("bv*+ba/b").
		Output("%(id)s.%(ext)s").
		OverallProgressFunc(time.Second, func(p ytdlp.OverallProgress) {
			approx := ""
			if p.TotalEstimated {
				approx = "~"
			}
			switch p.Phase {
			case ytdlp.ProgressPhaseDownloading:
				fmt.Printf("%s: %5.1f%% %d/%s%d bytes, format %d/%d (%s), %d B/s, eta %s\n", //nolint:forbidigo
					p.ID, p.Percent, p.Downloaded, approx, p.Total, p.FormatIndex, p.FormatCount, p.FormatID, p.Speed, p.ETA)
			case ytdlp.ProgressPhasePostProcessing:
				fmt.Printf("%s: post-processing (%s)\n", p.ID, p.PostProcessor) //nolint:forbidigo
			case ytdlp.ProgressPhaseFinished:
				fmt.Printf("%s: finished, %d bytes\n", p.ID, p.Total) //nolint:forbidigo
			}
		}).
		Run(context.TODO(), url)
	if err != nil {
		panic(err)
	}
}
