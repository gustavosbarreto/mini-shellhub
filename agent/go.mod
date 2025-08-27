module github.com/shellhub-io/mini-shellhub/agent

go 1.23.0

toolchain go1.24.6

require (
	github.com/gorilla/websocket v1.5.3
	github.com/labstack/echo/v4 v4.13.4
	github.com/shellhub-io/shellhub v0.20.0
	github.com/sirupsen/logrus v1.9.3
)

require (
	github.com/labstack/gommon v0.4.2 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.2 // indirect
)

require (
	github.com/GehirnInc/crypt v0.0.0-20230320061759-8cc1b52080c5 // indirect
	github.com/anmitsu/go-shlex v0.0.0-20200514113438-38f4b401e2be // indirect
	github.com/creack/pty v1.1.18 // indirect
	github.com/gliderlabs/ssh v0.3.5 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.11.2 // indirect
	github.com/go-resty/resty/v2 v2.7.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/jarcoal/httpmock v1.4.1 // indirect
	github.com/leodido/go-urn v1.2.2 // indirect
	github.com/openwall/yescrypt-go v1.0.0 // indirect
	golang.org/x/crypto v0.39.0 // indirect
	golang.org/x/net v0.41.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.26.0 // indirect
)

replace github.com/shellhub-io/shellhub => ../

replace github.com/gliderlabs/ssh => github.com/shellhub-io/ssh v0.0.0-20230224143412-edd48dfd6eea
