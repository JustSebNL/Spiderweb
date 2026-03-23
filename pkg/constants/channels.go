// Package constants provides shared constants across the codebase.
package constants

// internalChannels defines channels that are used for internal communication
// and should not be exposed to external users or recorded as last active channel.
var internalChannels = map[string]struct{}{
	"cli":      {},
	"system":   {},
	"subagent": {},
	"openclaw": {},
}

// IsInternalChannel returns true if the channel is an internal channel.
func IsInternalChannel(channel string) bool {
	_, found := internalChannels[channel]
	return found
}

type ValveState int

const (
	ValveStateReject            ValveState = 0
	ValveStateAccept            ValveState = 1
	ValveStateBusy              ValveState = 2
	ValveStateInterruptAccepted ValveState = 3
	ValveStateStop              ValveState = 4
	ValveStateSystemError       ValveState = 5
	ValveStateUnknown           ValveState = 6
)

func ValveStateString(state ValveState) string {
	switch state {
	case ValveStateReject:
		return "reject"
	case ValveStateAccept:
		return "accept"
	case ValveStateBusy:
		return "busy"
	case ValveStateInterruptAccepted:
		return "interrupt"
	case ValveStateStop:
		return "stop"
	case ValveStateSystemError:
		return "error"
	case ValveStateUnknown:
		return "unknown"
	default:
		return "unknown"
	}
}
