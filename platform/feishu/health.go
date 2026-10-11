package feishu

import (
	"errors"

	"github.com/chenhg5/cc-connect/core"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"
)

func channelErrorCode(err error) string {
	var clientError *larkws.ClientError
	if errors.As(err, &clientError) {
		switch clientError.Code {
		case larkws.AuthFailed:
			return "credentials"
		case larkws.Forbidden:
			return "permissions"
		}
	}
	return "network"
}

func reportChannel(state, errorCode string) {
	core.ReportHealth(core.HealthEvent{Component: "channel", State: state, ErrorCode: errorCode})
}

func channelHealthOptions(report func(string, string)) []larkws.ClientOption {
	return []larkws.ClientOption{
		larkws.WithOnReady(func() { report("online", "") }),
		larkws.WithOnReconnected(func() { report("online", "") }),
		larkws.WithOnReconnecting(func() { report("connecting", "") }),
		larkws.WithOnDisconnected(func() { report("offline", "") }),
		larkws.WithOnError(func(err error) { report("offline", channelErrorCode(err)) }),
	}
}
