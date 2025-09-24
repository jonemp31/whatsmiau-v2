package whatsmiau

type Status string

const (
	Connected      = "open"
	Connecting     = "connecting"
	QrCode         = "qr-code"
	Pairing        = "pairing"
	PairingPending = "pairing-pending"
	Closed         = "closed"
)
