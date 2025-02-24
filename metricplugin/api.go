package metricplugin

import (
	"time"

	bsmsg "github.com/ipfs/go-bitswap/message"
	pbmsg "github.com/ipfs/go-bitswap/message/pb"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/pkg/errors"
)

// A BitswapMessage is the type pushed to remote clients for recorded incoming
// Bitswap messages.
type BitswapMessage struct {
	// Wantlist entries sent with this message.
	WantlistEntries []bsmsg.Entry `json:"wantlist_entries"`

	// Whether the wantlist entries are a full new wantlist.
	FullWantList bool `json:"full_wantlist"`

	// Blocks sent with this message.
	Blocks []cid.Cid `json:"blocks"`

	// Block presence indicators sent with this message.
	BlockPresences []BlockPresence `json:"block_presences"`

	// Underlay addresses of the peer we were connected to when the message
	// was received.
	ConnectedAddresses []ma.Multiaddr `json:"connected_addresses"`
}

// A BlockPresence indicates the presence or absence of a block.
type BlockPresence struct {
	// Cid is the referenced CID.
	Cid cid.Cid `json:"cid"`

	// Type indicates the block presence type.
	Type BlockPresenceType `json:"block_presence_type"`
}

// BlockPresenceType is an enum for presence or absence notifications.
type BlockPresenceType int

// Block presence constants.
const (
	// Have indicates that the peer has the block.
	Have BlockPresenceType = 0
	// DontHave indicates that the peer does not have the block.
	DontHave BlockPresenceType = 1
)

// ConnectionEventType specifies the type of connection event.
type ConnectionEventType int

const (
	// Connected specifies that a connection was opened.
	Connected ConnectionEventType = 0
	// Disconnected specifies that a connection was closed.
	Disconnected ConnectionEventType = 1
)

// A ConnectionEvent is the type pushed to remote clients for recorded
// connection events.
type ConnectionEvent struct {
	// The multiaddress of the remote peer.
	Remote ma.Multiaddr `json:"remote"`

	// The type of this event.
	ConnectionEventType ConnectionEventType `json:"connection_event_type"`
}

// An EventSubscriber can handle events generated by the monitor.
type EventSubscriber interface {
	// ID should return a unique identifier.
	// The identifier is used to keep track of subscribers.
	// The identifier may be reused, but only after the old use has been
	// unsubscribed.
	ID() string

	// BitswapMessageReceived handles a Bitswap message that was recorded by the
	// monitor.
	// This must not block.
	BitswapMessageReceived(timestamp time.Time, peer peer.ID, msg BitswapMessage) error

	// ConnectionEventRecorded handles a connection event that was recorded by
	// the monitor.
	// This must not block.
	ConnectionEventRecorded(timestamp time.Time, peer peer.ID, connEvent ConnectionEvent) error
}

// ErrAlreadySubscribed is returned by Subscribe if the given EventSubscriber is
// already subscribed.
var ErrAlreadySubscribed = errors.New("already subscribed")

// The MonitoringAPI encompasses methods related to monitoring Bitswap traffic.
// These are served via the TCP pubsub mechanism.
type MonitoringAPI interface {
	// Subscribe adds a subscriber to the event subscription service.
	// Returns ErrAlreadySubscribed if the given subscriber is already subscribed.
	// An EventSubscriber that returns an error on one of the notification
	// methods will be removed from the list of subscribers.
	Subscribe(subscriber EventSubscriber) error

	// Unsubscribe removes a subscriber from the event subscription service.
	// It is safe to call this multiple times with the same subscriber.
	Unsubscribe(subscriber EventSubscriber)
}

// The RPCAPI is the interface for RPC-like method calls.
// These are served via HTTP, reliably.
type RPCAPI interface {
	// MonitoringAddresses returns listening addresses to connect to for
	// Bitswap monitoring.
	MonitoringAddresses() []string

	// Ping is a no-op.
	Ping()

	// BroadcastBitswapWant broadcasts WANT_(HAVE|BLOCK) requests for the given
	// CIDs to all connected peers that support Bitswap.
	// Which request type to send is chosen by the capabilities of the remote
	// peer.
	// This is sent as one message, which is either sent completely or fails.
	BroadcastBitswapWant(cids []cid.Cid) []BroadcastWantStatus

	// BroadcastBitswapCancel broadcasts CANCEL entries for the given CIDs to
	// all connected peers that support Bitswap.
	// This is sent as one message, which is either sent completely or fails.
	BroadcastBitswapCancel(cids []cid.Cid) []BroadcastCancelStatus

	// BroadcastBitswapWantCancel broadcasts WANT_(HAVE|BLOCK) requests for the
	// given CIDs, followed by CANCEL entries after a given time to all
	// connected peers that support Bitswap.
	BroadcastBitswapWantCancel(cids []cid.Cid, secondsBetween uint) []BroadcastWantCancelStatus
}

// PluginAPI describes the functionality provided by this monitor to remote
// clients.
type PluginAPI interface {
	MonitoringAPI

	RPCAPI
}

// BroadcastSendStatus contains basic information about a send operation to
// a single peer as part of a Bitswap broadcast.
type BroadcastSendStatus struct {
	TimestampBeforeSend time.Time `json:"timestamp_before_send"`
	SendDurationMillis  int64     `json:"send_duration_millis"`
	Error               error     `json:"error,omitempty"`
}

// BroadcastStatus contains additional basic information about a send operation
// to a single peer as part of a Bitswap broadcast.
type BroadcastStatus struct {
	BroadcastSendStatus
	Peer peer.ID `json:"peer"`
	// Underlay addresses of the peer we were connected to when the message
	// was sent, or empty if there was an error.
	ConnectedAddresses []ma.Multiaddr `json:"connected_addresses,omitempty"`
}

// BroadcastWantStatus describes the status of a send operation to a single
// peer as part of a Bitswap WANT broadcast.
type BroadcastWantStatus struct {
	BroadcastStatus
	RequestTypeSent *pbmsg.Message_Wantlist_WantType `json:"request_type_sent,omitempty"`
}

// BroadcastCancelStatus describes the status of a send operation to a single
// peer as part of a Bitswap CANCEL broadcast.
type BroadcastCancelStatus struct {
	BroadcastStatus
}

// BroadcastWantCancelWantStatus contains information about the send-WANT
// operation to a single peer as part of a Bitswap WANT+CANCEL broadcast.
type BroadcastWantCancelWantStatus struct {
	BroadcastSendStatus
	RequestTypeSent *pbmsg.Message_Wantlist_WantType `json:"request_type_sent,omitempty"`
}

// BroadcastWantCancelStatus describes the status of a send operation to a
// single peer as part of a Bitswap WANT+CANCEL broadcast.
type BroadcastWantCancelStatus struct {
	Peer               peer.ID        `json:"peer"`
	ConnectedAddresses []ma.Multiaddr `json:"connected_addresses,omitempty"`

	WantStatus   BroadcastWantCancelWantStatus `json:"want_status"`
	CancelStatus BroadcastSendStatus           `json:"cancel_status"`
}
