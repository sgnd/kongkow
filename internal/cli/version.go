package cli

import (
	"runtime"

	"github.com/spf13/cobra"

	"github.com/sgnd/kongkow/internal/out"
)

type versionInfo struct {
	Version string `json:"version"`
	Go      string `json:"go"`
	OS      string `json:"os"`
	Arch    string `json:"arch"`
}

func (v versionInfo) Text() string {
	return "kongkow " + v.Version + " (" + v.Go + " " + v.OS + "/" + v.Arch + ")\n"
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print kongkow's version",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			out.P.Print("version", versionInfo{
				Version: Version,
				Go:      runtime.Version(),
				OS:      runtime.GOOS,
				Arch:    runtime.GOARCH,
			}, nil)
			return nil
		},
	}
}
