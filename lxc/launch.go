package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gosexy/gettext"
	"github.com/lxc/lxd"
	"github.com/lxc/lxd/internal/gnuflag"
	"github.com/lxc/lxd/shared"
)

type launchCmd struct{}

func (c *launchCmd) showByDefault() bool {
	return true
}

func (c *launchCmd) usage() string {
	return gettext.Gettext(
		"lxc launch <image> [<name>] [--ephemeral|-e] [--profile|-p <profile>...]\n" +
			"\n" +
			"Launches a container using the specified image and name.\n" +
			"\n" +
			"Not specifying -p will result in the default profile.\n" +
			"Specifying \"-p\" with no argument will result in no profile.\n" +
			"\n" +
			"Example:\n" +
			"lxc launch ubuntu u1\n")
}

func (c *launchCmd) flags() {
	massage_args()
	gnuflag.Var(&profArgs, "profile", "Profile to apply to the new container")
	gnuflag.Var(&profArgs, "p", "Profile to apply to the new container")
	gnuflag.BoolVar(&ephem, "ephemeral", false, gettext.Gettext("Ephemeral container"))
	gnuflag.BoolVar(&ephem, "e", false, gettext.Gettext("Ephemeral container"))
}

func (c *launchCmd) run(config *lxd.Config, args []string) error {

	if len(args) > 2 || len(args) < 1 {
		return errArgs
	}

	iremote, image := config.ParseRemoteAndContainer(args[0])

	var name string
	var remote string
	if len(args) == 2 {
		remote, name = config.ParseRemoteAndContainer(args[1])
	} else {
		name = ""
		remote = ""
	}

	if ephem {
		fmt.Printf(gettext.Gettext("Ephemeral containers not yet supported\n"))
		return errArgs
	}

	fmt.Printf("Creating container...")
	d, err := lxd.NewClient(config, remote)
	if err != nil {
		return err
	}

	/*
	 * requested_empty_profiles means user requested empty
	 * !requested_empty_profiles but len(profArgs) == 0 means use profile default
	 */
	var resp *lxd.Response
	profiles := []string{}
	for _, p := range profArgs {
		profiles = append(profiles, p)
	}
	if !requested_empty_profiles && len(profiles) == 0 {
		resp, err = d.Init(name, iremote, image, nil)
	} else {
		resp, err = d.Init(name, iremote, image, &profiles)
	}
	if err != nil {
		if lxd.LXDErrors[http.StatusNotFound] == err {
			return fmt.Errorf("image doesn't exist")
		}
		return err
	}

	if name == "" {
		if resp.Resources == nil {
			return fmt.Errorf(gettext.Gettext("didn't get any affected resources from server"))
		}

		containers, ok := resp.Resources["containers"]
		if !ok || len(containers) == 0 {
			return fmt.Errorf(gettext.Gettext("didn't get any affected resources from server"))
		}

		var version string
		toScan := strings.Replace(containers[0], "/", " ", -1)
		count, err := fmt.Sscanf(toScan, " %s containers %s", &version, &name)
		if err != nil {
			return err
		}

		if count != 2 {
			return fmt.Errorf(gettext.Gettext("bad number of things scanned from resource"))
		}

		if version != shared.APIVersion {
			return fmt.Errorf(gettext.Gettext("got bad version"))
		}
	}

	if err = d.WaitForSuccess(resp.Operation); err != nil {
		return err
	}
	fmt.Println("done")

	fmt.Printf("Starting container...")
	resp, err = d.Action(name, shared.Start, -1, false)
	if err != nil {
		return err
	}

	err = d.WaitForSuccess(resp.Operation)
	fmt.Println("done")

	return err
}
