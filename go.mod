module github.com/coryschwartz/tgbridge

go 1.16

replace github.com/testground/testground => ../../testground/testground

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/go-git/go-git/v5 v5.0.0
	github.com/google/go-github/v33 v33.0.0
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79
	github.com/palantir/go-githubapp v0.6.0
	github.com/pkg/errors v0.9.1
	github.com/testground/testground v0.5.3
	gopkg.in/yaml.v2 v2.2.8
)
