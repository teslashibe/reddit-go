package redditmessenger

import "time"

// ChatIdentity is the Matrix-side identity of the user.
type ChatIdentity struct {
	UserID   string `json:"user_id"`
	DeviceID string `json:"device_id"`
	IsGuest  bool   `json:"is_guest"`
}

// ChatRoom represents a Reddit chat room (Matrix room).
type ChatRoom struct {
	RoomID string `json:"room_id"`
}

// ChatMember is a participant in a chat room.
type ChatMember struct {
	UserID      string `json:"user_id"`
	DisplayName string `json:"display_name"`
	Membership  string `json:"membership"`
	Username    string `json:"username,omitempty"`
}

// ChatMessage is a single message in a chat room.
type ChatMessage struct {
	EventID  string    `json:"event_id"`
	RoomID   string    `json:"room_id"`
	Sender   string    `json:"sender"`
	Body     string    `json:"body"`
	MsgType  string    `json:"msgtype"`
	Type     string    `json:"type"`
	Created  time.Time `json:"created"`
}

// ChatMessageListing is a paginated list of chat messages.
type ChatMessageListing struct {
	Messages []ChatMessage
	Start    string
	End      string
}

// matrixRoomListResponse is the raw response from /_matrix/client/r0/joined_rooms.
type matrixRoomListResponse struct {
	JoinedRooms []string `json:"joined_rooms"`
}

// matrixMessageResponse is the raw response from room messages endpoint.
type matrixMessageResponse struct {
	Start string        `json:"start"`
	End   string        `json:"end"`
	Chunk []matrixEvent `json:"chunk"`
}

// matrixEvent is a generic Matrix event.
type matrixEvent struct {
	Content         matrixContent `json:"content"`
	EventID         string        `json:"event_id"`
	OriginServerTS  int64         `json:"origin_server_ts"`
	RoomID          string        `json:"room_id"`
	Sender          string        `json:"sender"`
	Type            string        `json:"type"`
	StateKey        string        `json:"state_key,omitempty"`
	Unsigned        matrixUnsigned `json:"unsigned,omitempty"`
}

type matrixContent struct {
	Body        string `json:"body,omitempty"`
	MsgType     string `json:"msgtype,omitempty"`
	DisplayName string `json:"displayname,omitempty"`
	Membership  string `json:"membership,omitempty"`
}

type matrixUnsigned struct {
	Age       int64          `json:"age,omitempty"`
	Relations matrixRelation `json:"m.relations,omitempty"`
}

type matrixRelation struct {
	Profile matrixProfile `json:"com.reddit.profile,omitempty"`
}

type matrixProfile struct {
	IconURL  string `json:"icon_url,omitempty"`
	IsNSFW   bool   `json:"is_nsfw,omitempty"`
	Username string `json:"username,omitempty"`
}

type matrixMemberResponse struct {
	Chunk []matrixEvent `json:"chunk"`
}

type matrixCreateRoomRequest struct {
	IsDirect bool     `json:"is_direct"`
	Invite   []string `json:"invite"`
	Preset   string   `json:"preset"`
}

type matrixCreateRoomResponse struct {
	RoomID string `json:"room_id"`
}

type matrixSendResponse struct {
	EventID string `json:"event_id"`
}
