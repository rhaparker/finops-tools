module github.com/openshift-online/finops-tools/cli

go 1.24

require (
	github.com/openshift-online/finops-tools/core v0.0.0
	github.com/spf13/cobra v1.9.1
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
)

replace github.com/openshift-online/finops-tools/core => ../core
