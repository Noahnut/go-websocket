package websocket

var (
	webSocketString              = []byte("websocket")
	upgradeString                = []byte("Upgrade")
	connectionString             = []byte("Connection")
	websocketProtocolString      = []byte("Sec-WebSocket-Protocol")
	websocketKeyString           = []byte("Sec-WebSocket-Key")
	secWebsocketKeyValueString   = []byte("dGhlIHNhbXBsZSBub25jZQ==")
	websocketVersionString       = []byte("Sec-WebSocket-Version")
	websocketAcceptVersionString = []byte("13")
	websocketAcceptString        = []byte("Sec-WebSocket-Accept")
	getString                    = []byte("GET")
	originString                 = []byte("Origin")
)

type frameTypeCode uint8

const (
	codeContinuation frameTypeCode = 0x0

	codeText frameTypeCode = 0x1

	codeBinary frameTypeCode = 0x2

	codeClose frameTypeCode = 0x8

	codePing frameTypeCode = 0x9

	codePong frameTypeCode = 0xA

	codeUnknown frameTypeCode = 0xFF
)

func (f frameTypeCode) String() string {
	switch f {
	case codeContinuation:
		return "Continuation"
	case codeText:
		return "Text"
	case codeBinary:
		return "Binary"
	case codeClose:
		return "Close"
	case codePing:
		return "Ping"
	case codePong:
		return "Pong"
	default:
		return "Unknown"
	}
}

type websocketStatusCode uint16

const (
	websocketStatusCodeNormalClosure websocketStatusCode = 1000

	websocketStatusCodeGoingAway = 1001

	websocketStatusCodeProtocolError = 1002

	websocketStatusCodeUnsupportedData = 1003

	websocketStatusCodeNoStatusReceived = 1005

	websocketStatusCodeAbnormalClosure = 1006

	websocketStatusCodeInvalidFramePayloadData = 1007

	websocketStatusCodePolicyViolation = 1008

	websocketStatusCodeMessageTooBig = 1009

	websocketStatusCodeMandatoryExtension = 1010

	websocketStatusCodeInternalServerError = 1011
)

func (s websocketStatusCode) String() string {
	switch s {
	case websocketStatusCodeNormalClosure:
		return "NormalClosure"
	case websocketStatusCodeGoingAway:
		return "GoingAway"
	case websocketStatusCodeProtocolError:
		return "ProtocolError"
	case websocketStatusCodeUnsupportedData:
		return "UnsupportedData"
	case websocketStatusCodeNoStatusReceived:
		return "NoStatusReceived"
	case websocketStatusCodeAbnormalClosure:
		return "AbnormalClosure"
	case websocketStatusCodeInvalidFramePayloadData:
		return "InvalidFramePayloadData"
	case websocketStatusCodePolicyViolation:
		return "PolicyViolation"
	case websocketStatusCodeMessageTooBig:
		return "MessageTooBig"
	case websocketStatusCodeMandatoryExtension:
		return "MandatoryExtension"
	case websocketStatusCodeInternalServerError:
		return "InternalServerError"
	default:
		return "Unknown"
	}
}
