// This file is AI generated from Platform SDK.

package main

import "go.mau.fi/util/jsontime"

type ID = string
type AttachmentSize struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type AttachmentType string

const (
	AttachmentTypeUnknown AttachmentType = "unknown"
	AttachmentTypeImg     AttachmentType = "img"
	AttachmentTypeVideo   AttachmentType = "video"
	AttachmentTypeAudio   AttachmentType = "audio"
)

type AttachmentPlayStatus string

const (
	PlayStatusUnplayed AttachmentPlayStatus = "UNPLAYED"
	PlayStatusPlayed   AttachmentPlayStatus = "PLAYED"
)

type AttachmentID = string

type AttachmentBase struct {
	ID          AttachmentID         `json:"id"`
	Type        AttachmentType       `json:"type"`
	Size        *AttachmentSize      `json:"size,omitempty"`
	PosterImg   string               `json:"posterImg,omitempty"`
	MimeType    string               `json:"mimeType,omitempty"`
	FileName    string               `json:"fileName,omitempty"`
	FileSize    int                  `json:"fileSize,omitempty"`
	Loading     bool                 `json:"loading,omitempty"`
	IsGif       bool                 `json:"isGif,omitempty"`
	IsSticker   bool                 `json:"isSticker,omitempty"`
	IsVoiceNote bool                 `json:"isVoiceNote,omitempty"`
	PlayStatus  AttachmentPlayStatus `json:"playStatus,omitempty"`
	Extra       interface{}          `json:"extra,omitempty"`
}

type AttachmentWithURL struct {
	AttachmentBase
	SrcURL string `json:"srcURL"`
}

type AttachmentWithBuffer struct {
	AttachmentBase
	Data []byte `json:"data"`
}

type Attachment interface{} // Can be AttachmentWithURL or AttachmentWithBuffer

type MessageID = string

type MessageReaction struct {
	ID            string `json:"id"`
	ReactionKey   string `json:"reactionKey"`
	ImgURL        string `json:"imgURL,omitempty"`
	ParticipantID string `json:"participantID"`
	Emoji         bool   `json:"emoji,omitempty"`
}

type MessageSeen interface{} // Can be bool, time.Time, or map[string]interface{}

type MessagePreview struct {
	ID          string       `json:"id"`
	ThreadID    string       `json:"threadID,omitempty"`
	Text        string       `json:"text,omitempty"`
	SenderID    string       `json:"senderID"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

type MessageLink struct {
	URL         string          `json:"url"`
	OriginalURL string          `json:"originalURL,omitempty"`
	Favicon     string          `json:"favicon,omitempty"`
	Img         string          `json:"img,omitempty"`
	ImgSize     *AttachmentSize `json:"imgSize,omitempty"`
	Title       string          `json:"title"`
	Summary     string          `json:"summary,omitempty"`
}

type MessageButton struct {
	Label   string `json:"label"`
	LinkURL string `json:"linkURL"`
}

type RoomInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

type Message struct {
	ID               string              `json:"id"`
	Timestamp        jsontime.UnixMilli  `json:"timestamp"`
	EditedTimestamp  *jsontime.UnixMilli `json:"editedTimestamp,omitempty"`
	ExpiresInSeconds *int                `json:"expiresInSeconds,omitempty"`
	ForwardedCount   *int                `json:"forwardedCount,omitempty"`
	ForwardedFrom    interface{}         `json:"forwardedFrom,omitempty"`
	SenderID         string              `json:"senderID"`

	Text           string      `json:"text,omitempty"`
	TextAttributes interface{} `json:"textAttributes,omitempty"`
	TextHeading    string      `json:"textHeading,omitempty"`
	TextFooter     string      `json:"textFooter,omitempty"`

	Attachments []Attachment  `json:"attachments,omitempty"`
	Tweets      []interface{} `json:"tweets,omitempty"`
	Links       []MessageLink `json:"links,omitempty"`
	IframeURL   string        `json:"iframeURL,omitempty"`

	Reactions     []MessageReaction `json:"reactions,omitempty"`
	Seen          MessageSeen       `json:"seen,omitempty"`
	IsDelivered   bool              `json:"isDelivered,omitempty"`
	IsHidden      bool              `json:"isHidden,omitempty"`
	IsSender      bool              `json:"isSender,omitempty"`
	IsAction      bool              `json:"isAction,omitempty"`
	IsDeleted     bool              `json:"isDeleted,omitempty"`
	IsErrored     bool              `json:"isErrored,omitempty"`
	ParseTemplate bool              `json:"parseTemplate,omitempty"`

	LinkedMessageThreadID string          `json:"linkedMessageThreadID,omitempty"`
	LinkedMessageID       string          `json:"linkedMessageID,omitempty"`
	LinkedMessage         *MessagePreview `json:"linkedMessage,omitempty"`
	Action                interface{}     `json:"action,omitempty"`
	Buttons               []MessageButton `json:"buttons,omitempty"`

	Behavior string `json:"behavior,omitempty"`

	RoomInfo *RoomInfo `json:"roomInfo,omitempty"`

	SortKey interface{} `json:"sortKey,omitempty"`

	Extra    interface{} `json:"extra,omitempty"`
	Original string      `json:"_original,omitempty"`
	Cursor   string      `json:"cursor,omitempty"`

	URL string `json:"url,omitempty"`
}
