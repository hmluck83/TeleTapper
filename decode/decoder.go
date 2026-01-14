package decode

import (
	"fmt"

	"github.com/go-faster/errors"
	"github.com/gotd/td/bin"
	"github.com/gotd/td/mt"
	"github.com/gotd/td/proto"
)

func HandleContainer(msgID int64, b *bin.Buffer) error {
	var container proto.MessageContainer
	if err := container.Decode(b); err != nil {
		return errors.Wrap(err, "container")
	}
	for _, msg := range container.Messages {
		b := &bin.Buffer{Buf: msg.Body}
		HandleMessage(msgID, b)
	}
	return nil
}

func HandleMessage(msgID int64, buf *bin.Buffer) {
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
		if err := HandleContainer(msgID, buf); err != nil {
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
