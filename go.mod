module github.com/terraform-providers/terraform-provider-aws

go 1.16

require (
	github.com/agl/ed25519 v0.0.0-20170116200512-5312a6153412 // indirect
	github.com/aws/aws-sdk-go v1.40.40
	github.com/beevik/etree v1.1.0
	github.com/fatih/color v1.9.0 // indirect
	github.com/hashicorp/aws-sdk-go-base v0.7.1
	github.com/hashicorp/go-cleanhttp v0.5.2
	github.com/hashicorp/go-cty v1.4.1-0.20200414143053-d3edf31b6320
	github.com/hashicorp/go-multierror v1.1.1
	github.com/hashicorp/go-version v1.3.0
	github.com/hashicorp/terraform-plugin-sdk/v2 v2.7.1
	github.com/jen20/awspolicyequivalence v1.1.0
	github.com/keybase/go-crypto v0.0.0-20161004153544-93f5b35093ba
	github.com/mattn/go-colorable v0.1.7 // indirect
	github.com/mitchellh/copystructure v1.2.0
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mitchellh/go-testing-interface v1.14.1
	github.com/pquerna/otp v1.3.0
	github.com/shopspring/decimal v1.2.0
	golang.org/x/crypto v0.0.0-20210421170649-83a5a9bb288b
	gopkg.in/yaml.v2 v2.4.0
)

replace github.com/hashicorp/terraform-plugin-sdk/v2 => github.com/gdavison/terraform-plugin-sdk/v2 v2.0.2-0.20210714181518-b5a3dc95a675
