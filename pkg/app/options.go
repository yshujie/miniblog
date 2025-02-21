package app

import (
	cliflag "github.com/marmotedu/component-base/pkg/cli/flag"
)

type CliOptions interface {
	// AddFlags adds flags to the specified FlagSet object.
	// AddFlags(fs *pflag.FlagSet)
	Flags() (fss cliflag.NamedFlagSets)
	Validate() []error
}

// CompleteableOptions abstracts options which can be completed.
type CompleteableOptions interface {
	Complete() error
}
