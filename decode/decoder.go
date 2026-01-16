package decode

import (
	"fmt"

	"github.com/go-faster/errors"
	"github.com/gotd/td/bin"
	"github.com/gotd/td/mt"
	"github.com/gotd/td/proto"
	"github.com/gotd/td/tg"
)

func HandleContainer(msgID int64, b *bin.Buffer, direction string) error {
	var container proto.MessageContainer
	if err := container.Decode(b); err != nil {
		return errors.Wrap(err, "container")
	}
	for _, msg := range container.Messages {
		b := &bin.Buffer{Buf: msg.Body}
		HandleMessage(msgID, b, direction)
	}
	return nil
}

func HandleMessage(msgID int64, buf *bin.Buffer, direction string) {
	id, err := buf.PeekID()
	if err != nil {
		fmt.Printf("  Failed to read message ID: %v\n", err)
		return
	}
	switch id {
	case mt.NewSessionCreatedTypeID:
		fmt.Println("New Session Created")
	case mt.BadMsgNotificationTypeID, mt.BadServerSaltTypeID:
		fmt.Println("Bad Message Notification")
	case mt.FutureSaltsTypeID:
		fmt.Println("Future Salts")
	case proto.MessageContainerTypeID:
		fmt.Println("Message Container")
		if err := HandleContainer(msgID, buf, direction); err != nil {
			fmt.Printf("  Failed to handle container: %v\n", err)
		}
	case proto.ResultTypeID:
		fmt.Println("Result Message")
	case mt.PongTypeID:
		fmt.Println("Pong Message")
	case mt.MsgsAckTypeID:
		fmt.Println("Messages Acknowledgment")
	case proto.GZIPTypeID:
		fmt.Println("GZIP Compressed Message")
	case mt.MsgDetailedInfoTypeID:
		fmt.Println("Message Detailed Info")
	default:
		if direction == "receive" {
			if err := handleUpdate(msgID, buf); err != nil {
				fmt.Printf("  Failed to handle update: %v\n", err)
			}
			return
		}

		id, err := buf.PeekID()
		if err != nil {
			fmt.Printf("  Failed to read message ID: %v\n", err)
			return
		}

		predicate, ok := ConstructorMap[id]
		if ok {
			fmt.Printf("  Message Type: %s\n", predicate)
		} else {
			fmt.Printf("  Unknown Message Type: 0x%08x\n", buf.Buf)
			fmt.Println(string(buf.Buf))
		}
	}
}

func handleUpdate(msgID int64, b *bin.Buffer) error {
	updateClass, err := tg.DecodeUpdates(b)
	if err != nil {
		return errors.Wrap(err, "decode update class")
	}
	switch u := updateClass.(type) {
	case *tg.UpdateShortMessage:
		fmt.Printf("  Short Message: %q, id: %d\n", u.Message, u.ID)
	case *tg.UpdateShortChatMessage:
		fmt.Printf("  Short Chat Message: %q, id: %d\n", u.Message, u.ID)
	default:
		fmt.Printf("  Update type: %T\n", u)
	}
	return nil
}
