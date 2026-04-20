package reddit

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// ChatWhoAmI returns the Matrix-side identity of the authenticated user.
func (m *Client) ChatWhoAmI() (*ChatIdentity, error) {
	var id ChatIdentity
	if err := m.matrixGetJSON("/_matrix/client/r0/account/whoami", &id); err != nil {
		return nil, err
	}
	return &id, nil
}

// ChatRooms returns all chat rooms the user has joined.
func (m *Client) ChatRooms() ([]ChatRoom, error) {
	var resp matrixRoomListResponse
	if err := m.matrixGetJSON("/_matrix/client/r0/joined_rooms", &resp); err != nil {
		return nil, err
	}

	rooms := make([]ChatRoom, len(resp.JoinedRooms))
	for i, id := range resp.JoinedRooms {
		rooms[i] = ChatRoom{RoomID: id}
	}
	return rooms, nil
}

// ChatMessages returns the most recent messages from a chat room.
// Direction is backwards (newest first). Use the returned End token
// to paginate further back with ChatMessagesFrom.
func (m *Client) ChatMessages(roomID string, limit int) (*ChatMessageListing, error) {
	return m.ChatMessagesFrom(roomID, limit, "")
}

// ChatMessagesFrom returns messages starting from the given pagination token.
func (m *Client) ChatMessagesFrom(roomID string, limit int, from string) (*ChatMessageListing, error) {
	encoded := url.PathEscape(roomID)
	path := "/_matrix/client/r0/rooms/" + encoded + "/messages?dir=b"
	if limit > 0 {
		path += "&limit=" + strconv.Itoa(limit)
	}
	if from != "" {
		path += "&from=" + url.QueryEscape(from)
	}

	var resp matrixMessageResponse
	if err := m.matrixGetJSON(path, &resp); err != nil {
		return nil, err
	}

	listing := &ChatMessageListing{
		Start: resp.Start,
		End:   resp.End,
	}
	for _, ev := range resp.Chunk {
		if ev.Type != "m.room.message" {
			continue
		}
		listing.Messages = append(listing.Messages, ChatMessage{
			EventID: ev.EventID,
			RoomID:  ev.RoomID,
			Sender:  ev.Sender,
			Body:    ev.Content.Body,
			MsgType: ev.Content.MsgType,
			Type:    ev.Type,
			Created: time.UnixMilli(ev.OriginServerTS),
		})
	}
	return listing, nil
}

// ChatMembers returns the members of a chat room.
func (m *Client) ChatMembers(roomID string) ([]ChatMember, error) {
	encoded := url.PathEscape(roomID)
	var resp matrixMemberResponse
	if err := m.matrixGetJSON("/_matrix/client/r0/rooms/"+encoded+"/members", &resp); err != nil {
		return nil, err
	}

	var members []ChatMember
	for _, ev := range resp.Chunk {
		if ev.Type != "m.room.member" {
			continue
		}
		cm := ChatMember{
			UserID:      ev.StateKey,
			DisplayName: ev.Content.DisplayName,
			Membership:  ev.Content.Membership,
		}
		if ev.Unsigned.Relations.Profile.Username != "" {
			cm.Username = ev.Unsigned.Relations.Profile.Username
		}
		members = append(members, cm)
	}
	return members, nil
}

// ChatSend sends a text message to a chat room.
// Returns the event ID of the sent message.
func (m *Client) ChatSend(roomID, body string) (string, error) {
	encoded := url.PathEscape(roomID)
	txnID := fmt.Sprintf("m%d.%d", time.Now().UnixMilli(), time.Now().UnixNano()%1000)
	path := "/_matrix/client/r0/rooms/" + encoded + "/send/m.room.message/" + txnID

	payload := map[string]string{
		"msgtype": "m.text",
		"body":    body,
	}

	respBody, err := m.matrixPut(path, payload)
	if err != nil {
		return "", err
	}

	var resp matrixSendResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return "", fmt.Errorf("decoding send response: %w", err)
	}
	return resp.EventID, nil
}

// ChatCreateDM creates a direct-message room with the given Matrix user ID.
// Matrix user IDs look like "@t2_xxxxx:reddit.com".
// Returns the new room.
func (m *Client) ChatCreateDM(matrixUserID string) (*ChatRoom, error) {
	req := matrixCreateRoomRequest{
		IsDirect: true,
		Invite:   []string{matrixUserID},
		Preset:   "trusted_private_chat",
	}

	respBody, err := m.matrixPost("/_matrix/client/r0/createRoom", req)
	if err != nil {
		return nil, err
	}

	var resp matrixCreateRoomResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("decoding create room response: %w", err)
	}
	return &ChatRoom{RoomID: resp.RoomID}, nil
}
