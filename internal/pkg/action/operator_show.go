package action

import (
	"context"
	"fmt"
	"time"

	operatorsv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/package-server/apis/operators/v1"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/types"
)

type OperatorShow struct {
	config *Configuration

	Package         string
	Channel         string
	LongDescription bool
	ShowTimeout     time.Duration

	Logf func(string, ...interface{})
}

func NewOperatorShow(cfg *Configuration) *OperatorShow {
	return &OperatorShow{
		config: cfg,
		Logf:   func(string, ...interface{}) {},
	}
}

func (s *OperatorShow) BindFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&s.Channel, "channel", "c", "", "channel")
	fs.BoolVarP(&s.LongDescription, "with-long-description", "L", false, "long description")
	fs.DurationVarP(&s.ShowTimeout, "timeout", "t", time.Minute, "the amount of time to wait before cancelling the show request")

}

var (
	pkgHdr  = asHeader("Package")
	repoHdr = asHeader("Repository")
	catHdr  = asHeader("Catalog")
	chHdr   = asHeader("Channels")
	imHdr   = asHeader("Install Modes")
	sdHdr   = asHeader("Description")
	ldHdr   = asHeader("Long Description")

	repoAnnot = "repository"
	descAnnot = "description"
)

func (s *OperatorShow) Run(ctx context.Context) ([]string, error) {
	out := make([]string, 0)

	// get packagemanifest for provided package name
	pm, err := s.getPackageManifest(ctx)
	if err != nil {
		return nil, fmt.Errorf("get package manifest: %v", err)
	}

	// determine channel from flag or default
	pc, err := s.getPackageChannel(pm)
	if err != nil {
		return nil, fmt.Errorf("get package channel: %v", err)
	}

	// prepare output to return
	out = append(out,
		// package
		pkgHdr+fmt.Sprintf("%s %s (by %s)\n\n",
			pc.CurrentCSVDesc.DisplayName,
			pc.CurrentCSVDesc.Version,
			pc.CurrentCSVDesc.Provider.Name),
		// repo
		repoHdr+fmt.Sprintf("%s\n\n",
			pc.CurrentCSVDesc.Annotations[repoAnnot]),
		// catalog
		catHdr+fmt.Sprintf("%s\n\n", pm.Status.CatalogSourceDisplayName),
		// available channels
		chHdr+fmt.Sprintf("%s\n\n",
			asNewlineDelimString(s.getAvailableChannels(pm))),
		// install modes
		imHdr+fmt.Sprintf("%s\n\n",
			asNewlineDelimString(s.getSupportedInstallModes(pc))),
		// description
		sdHdr+fmt.Sprintf("%s\n",
			pc.CurrentCSVDesc.Annotations[descAnnot]),
	)

	if s.LongDescription {
		out = append(out,
			"\n"+ldHdr+pm.Status.Channels[0].CurrentCSVDesc.LongDescription)
	}

	return out, nil
}

// getPackageManifest returns the packagemanifest that matches the namespace and package arguments
// from the API server.
// TODO(): This is pretty much identical to OperatorInstall's getPackageChannel. Might be worth consolidating.
func (s *OperatorShow) getPackageManifest(ctx context.Context) (*operatorsv1.PackageManifest, error) {
	pm := &operatorsv1.PackageManifest{}
	key := types.NamespacedName{
		Namespace: s.config.Namespace,
		Name:      s.Package,
	}
	if err := s.config.Client.Get(ctx, key, pm); err != nil {
		return nil, err
	}
	return pm, nil
}

// getPackageChannel returns the package channel specified, or the default if none was specified.
// TODO(): This is pretty much identical to OperatorInstall's getPackageChannel. Might be worth consolidating.
func (s *OperatorShow) getPackageChannel(pm *operatorsv1.PackageManifest) (*operatorsv1.PackageChannel, error) {
	if s.Channel == "" {
		s.Channel = pm.Status.DefaultChannel
	}
	var packageChannel *operatorsv1.PackageChannel
	for _, ch := range pm.Status.Channels {
		ch := ch
		if ch.Name == s.Channel {
			packageChannel = &ch
		}
	}
	if packageChannel == nil {
		return nil, fmt.Errorf("channel %q does not exist for package %q", s.Channel, s.Package)
	}
	return packageChannel, nil
}

// getAvailableChannels lists all available package channels for the operator.
func (s *OperatorShow) getAvailableChannels(pm *operatorsv1.PackageManifest) []string {
	channels := make([]string, len(pm.Status.Channels))
	for i, channel := range pm.Status.Channels {
		n := channel.Name
		if channel.IsDefaultChannel(*pm) {
			n += " (default)"
		}
		if s.Channel == channel.Name {
			n += " (shown)"
		}
		channels[i] = n
	}

	return channels
}

// getSupportedInstallModes returns a string slice representation of install mode
// objects the operator supports.
func (s *OperatorShow) getSupportedInstallModes(pc *operatorsv1.PackageChannel) []string {
	supportedInstallModes := make([]string, 1)
	for _, imode := range pc.CurrentCSVDesc.InstallModes {
		if imode.Supported {
			supportedInstallModes = append(supportedInstallModes, string(imode.Type))
		}
	}

	return supportedInstallModes
}

// asNewlineDelimString returns a string slice as a single string
// separated by newlines
func asNewlineDelimString(stringItems []string) string {
	var res string
	for _, v := range stringItems {
		if res != "" {
			res = fmt.Sprintf("%s\n%s", res, v)
			continue
		}

		res = v
	}
	return res
}

// asHeader returns the string with "header bars" for displaying in
// plain text cases.
func asHeader(s string) string {
	return fmt.Sprintf("== %s ==\n", s)
}
