package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/pflag"

	"k8s.io/kubectl/pkg/util/templates"
	"k8s.io/kubernetes/test/e2e/upgrades"

	"github.com/openshift/origin/pkg/test/ginkgo"
	"github.com/openshift/origin/test/e2e/upgrade"
)

// upgradeSuites are all known upgade test suites this binary should run
var upgradeSuites = []*ginkgo.TestSuite{
	{
		Name: "all",
		Description: templates.LongDesc(`
		Run all tests.
		`),
		Matches: func(name string) bool {
			return strings.Contains(name, "[Feature:ClusterUpgrade]") && strings.Contains(name, "[Suite:openshift]")
		},
		Init: func(opt map[string]string) error {
			return upgradeInitArguments(opt, func(string) bool { return true })
		},
		TestTimeout: 240 * time.Minute,
	},
	{
		Name: "platform",
		Description: templates.LongDesc(`
		Run only the tests that verify the platform remains available.
		`),
		Matches: func(name string) bool {
			return strings.Contains(name, "[Feature:ClusterUpgrade]") && strings.Contains(name, "[Suite:openshift]")
		},
		Init: func(opt map[string]string) error {
			return upgradeInitArguments(opt, func(name string) bool { return name == "control-plane-available" })
		},
		TestTimeout: 240 * time.Minute,
	},
}

func upgradeInitArguments(opt map[string]string, filterFn func(name string) bool) error {
	for k, v := range opt {
		switch k {
		case "abort-at":
			if err := upgrade.SetUpgradeAbortAt(v); err != nil {
				return err
			}
		case "disrupt-reboot":
			if err := upgrade.SetUpgradeDisruptReboot(v); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unrecognized upgrade option: %s", k)
		}
	}
	return filterUpgrade(upgrade.AllTests(), filterFn)
}

type UpgradeOptions struct {
	Suite    string
	ToImage  string
	JUnitDir string

	TestOptions []string
}

func (o *UpgradeOptions) OptionsMap() (map[string]string, error) {
	options := make(map[string]string)
	for _, option := range o.TestOptions {
		parts := strings.SplitN(option, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("test option %q is not valid, must be KEY=VALUE", option)
		}
		if len(parts[0]) == 0 {
			return nil, fmt.Errorf("test option %q is not valid, must be KEY=VALUE", option)
		}
		_, exists := options[parts[0]]
		if exists {
			return nil, fmt.Errorf("option %q declared twice", parts[0])
		}
		options[parts[0]] = parts[1]
	}
	return options, nil
}

func (o *UpgradeOptions) ToEnv() string {
	out, err := json.Marshal(o)
	if err != nil {
		panic(err)
	}
	return string(out)
}

func initUpgrade(value string) (*UpgradeOptions, error) {
	var opt UpgradeOptions
	if len(value) == 0 {
		return &opt, nil
	}
	if err := json.Unmarshal([]byte(value), &opt); err != nil {
		return nil, err
	}
	for _, suite := range upgradeSuites {
		if suite.Name == opt.Suite {
			o, err := opt.OptionsMap()
			if err != nil {
				return nil, err
			}
			if suite.Init != nil {
				if err := suite.Init(o); err != nil {
					return nil, err
				}
			}
			upgrade.SetToImage(opt.ToImage)
			return &opt, nil
		}
	}
	return nil, fmt.Errorf("unrecognized upgrade info")
}

func filterUpgrade(tests []upgrades.Test, match func(string) bool) error {
	var scope []upgrades.Test
	for _, test := range tests {
		if match(test.Name()) {
			scope = append(scope, test)
		}
	}
	upgrade.SetTests(scope)
	return nil
}

func bindUpgradeOptions(opt *UpgradeOptions, flags *pflag.FlagSet) {
	flags.StringVar(&opt.ToImage, "to-image", opt.ToImage, "Specify the image to test an upgrade to.")
	flags.StringSliceVar(&opt.TestOptions, "options", opt.TestOptions, "A set of KEY=VALUE options to control the test. See the help text.")
}
